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

// The map and guard that we use to keep track of what
// sensors we've seen and their timestamps
var sensorMap map[string]time.Time
var mutex = &sync.Mutex{}

// The time, in seconds, of how long an 'active' status message
// can live before it's expired
const expiry = 15

// This function goes through the sensorMap every five seconds and
// checks to see what messages have expired (i.e. their timestamps
// are older than the expiry constant above). It builds a message line
// with either a 0 or 1 at the end to indicate that the sensor message
// has expired (i.e. there's no one there) or that there is still someone
// there, respectively
func buildTimeline() {
	for {
		newLine := ""
		time.Sleep(1 * time.Second)
		now := time.Now()
		if len(sensorMap) > 0 {
			mutex.Lock()
			for k, v := range sensorMap {
				// compare the current timestamp to that
				// in the map...
				diff := now.Sub(v)
				seconds := int(diff.Seconds())
				// has the message expired?
				if seconds > expiry {
					// Yes, so remove it from the map and...
					delete(sensorMap, k)
					// ...send the line with 0
					newLine = strconv.FormatInt(v.Unix(), 10) + "," + k + ",0"
				} else {
					// No, the message is still alive
					newLine = strconv.FormatInt(v.Unix(), 10) + "," + k + ",1"
				}

				// And send the message on its merry way
				fullStatusChannel <- newLine
			}
			mutex.Unlock()
		}
	}
}

func sendFullStatusMessage() {
	for {
		fullStatusMsg := <-fullStatusChannel
		fmt.Println("Gonna send this: ", fullStatusMsg)
		publishToTopic(fullStatusMsg)
	}
}

func main() {
	// The channel we're going to receive messages on
	statusChannel = make(chan StatusMessage)
	// The channel we're going to send the full data on
	fullStatusChannel = make(chan string)

	// Start up our publishing connection
	setupToPublish()

	// Set us up to listen to the topics on the MQTT server...
	go listenOnTopic()

	// Create our map that will hold the key of sensor name
	// to its timestamp
	sensorMap = make(map[string]time.Time)
	// This goroutine reads from the map
	go buildTimeline()
	// This goroutine sends the new status message to the MQTT server
	go sendFullStatusMessage()

	// And here we go!
	for {
		// Get our message from the MQTT topic
		statusMsg := <-statusChannel

		// And split it up into its various parts
		lineParts := strings.Split(statusMsg.spaceStatus, ",")
		i, err := strconv.ParseInt(lineParts[0], 10, 64)
		if err != nil {
			panic(err)
		}
		// Convert the unix timestamp to a time object for the map
		tm := time.Unix(i, 0)
		mutex.Lock()
		// Check to see if there's already an entry for this sensor in the map
		if _, testIfExists := sensorMap[lineParts[1]]; testIfExists {
			// There is an entry, but it has an older timestamp
			// so we are going to simply delete it, because we know that
			// the timestamp we have right now is newer than this one (even if
			// by a second), and that's all we care about
			delete(sensorMap, lineParts[1])
		}
		// Now we add our entry into the map so the buildTimeLine() function
		// can evaulate it
		sensorMap[lineParts[1]] = tm
		mutex.Unlock()
	}

	// Should never get here
	fmt.Println("Hello world!")
}
