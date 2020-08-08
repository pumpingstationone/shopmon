package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*
 * This program is basically a way to tell when we should stop paying
 * attention to a sensor. We only get notifications about when a sensor
 * is activated, but we don't have any way to keep track of when the sensor
 * is effectively off; we don't get any notification of that. So what we
 * do here is keep track of the messages we're getting and, after a certain
 * period of time, if we do not have any new messages for a particular sensor,
 * we say that sensor is "off" and sned a message indicating that.
 */

type StatusMessage struct {
	spaceStatus string
}

// Our channel that accepts StatusMessages from the MQTT server
var statusChannel chan StatusMessage

// The channel we are going to use to send the new messages to
// the MQTT server
var fullStatusChannel chan string

var sensorMap map[string]time.Time
var mutex = &sync.Mutex{}

func buildTimeline() {
	for {
		newLine := ""
		fmt.Println("Map size is", len(sensorMap))
		time.Sleep(5 * time.Second)
		now := time.Now()
		if len(sensorMap) > 0 {
			mutex.Lock()
			for k, v := range sensorMap {
				fmt.Println(k, "->", v)
				diff := now.Sub(v)
				seconds := int(diff.Seconds())
				fmt.Println("Seconds diff is ", seconds)
				if seconds > 30 {
					delete(sensorMap, k)
					// send the line with 0
					newLine = strconv.FormatInt(v.Unix(), 10) + "," + k + ",0"
				} else {
					// send the line with
					newLine = strconv.FormatInt(v.Unix(), 10) + "," + k + ",1"
				}

			}
			mutex.Unlock()
			// And send the message on its merry way
			fullStatusChannel <- newLine
		}
	}
}

func sendFullStatusMessage() {
	for {
		fullStatusMsg := <-fullStatusChannel
		fmt.Println("Gonna send this: ", fullStatusMsg)
	}
}

func main() {
	// The channel we're going to receive messages on
	statusChannel = make(chan StatusMessage)
	// The channel we're going to send the full data on
	fullStatusChannel = make(chan string)

	// Set us up to listen to the topics on the MQTT server...
	go listenOnTopic()

	sensorMap = make(map[string]time.Time)
	go buildTimeline()
	go sendFullStatusMessage()

	for {
		statusMsg := <-statusChannel
		fmt.Println("Oh yeah got", statusMsg.spaceStatus)

		lineParts := strings.Split(statusMsg.spaceStatus, ",")
		i, err := strconv.ParseInt(lineParts[0], 10, 64)
		if err != nil {
			panic(err)
		}
		tm := time.Unix(i, 0)
		mutex.Lock()
		if _, testIfExists := sensorMap[lineParts[1]]; testIfExists {
			fmt.Println("Deleting", lineParts[1])
			delete(sensorMap, lineParts[1])
		}
		fmt.Println("adding...")
		sensorMap[lineParts[1]] = tm
		mutex.Unlock()
	}

	fmt.Println("Hello world!")
}
