package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nlopes/slack"
)

// StatusMessage is a struct that is passed from the
// MQTT goroutine (see listenOnTopic() in mqtt.go)
type StatusMessage struct {
	spaceStatus string
}

// Our channel that accepts StatusMessages
var statusChannel chan StatusMessage

// The map and guard that we use to keep track of what
// areas we've seen and their timestamps
var sensorMap map[string]time.Time
var mutex = &sync.Mutex{}

func trimSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

func formatSeconds(duration time.Duration) string {
	timeLine := ""

	days := int(duration.Hours() / 24)
	if days > 0 {
		if days > 1 {
			timeLine = fmt.Sprintf("%d days, ", days)
		} else {
			timeLine = fmt.Sprintf("%d day, ", days)
		}
	}

	hours := int(duration.Hours())
	if days >= 1 {
		hours = int(duration.Hours() / (24.0 * float64(days)))
	}
	if hours > 0 {
		if hours > 1 {
			timeLine += fmt.Sprintf("%d hours, ", hours)
		} else {
			timeLine += fmt.Sprintf("%d hour, ", hours)
		}
	}

	minutes := int(duration.Minutes())
	if hours >= 1 {
		minutes = int(duration.Minutes() / (60.0 * float64(hours)))
	}
	if minutes > 0 {
		if minutes > 1 {
			timeLine += fmt.Sprintf("%d minutes, ", minutes)
		} else {
			timeLine += fmt.Sprintf("%d minute, ", minutes)
		}
	}

	seconds := int(duration.Seconds())
	if minutes >= 1 {
		seconds = int(duration.Seconds() / (60.0 * float64(minutes)))
	}
	fmt.Println("Seconds is: ", seconds)
	if seconds > 0 {
		if seconds > 1 {
			timeLine += fmt.Sprintf("%d seconds, ", seconds)
		} else {
			timeLine += fmt.Sprintf("%d second, ", seconds)
		}
	}

	return trimSuffix(strings.TrimSpace(timeLine), ",")
}

func reportForArea(input string) string {
	message := ""

	// We're looking for a message in the form of !area <area name>
	// and if we don't find that, we'll inform them of the areas we
	// do know about (this is read from the map so it can be kept dynamic
	// as more areas come online)

	area := ""
	areaParts := strings.Split(input, "!area")
	if len(areaParts) > 1 {
		area = strings.TrimSpace(areaParts[1])
		fmt.Println("The area we want is:", area)
	}

	// These two strings are for building the help message in case
	// we didn't get an area or an unknown area
	helpMsg := "Hmm, you want to enter `!area <area>` (case insensitive).\n_I currently know of the following areas:_ "
	areaList := ""

	// If we did find the area, this will tell us when it was last occupied
	areaStatus := ""

	now := time.Now()

	foundArea := false
	mutex.Lock()
	for k, v := range sensorMap {
		if strings.ToLower(k) == strings.ToLower(area) {
			diff := now.Sub(v)
			timeInfo := formatSeconds(diff)
			areaStatus = fmt.Sprintf("There was someone in `%s` *%s* ago", k, timeInfo)
			foundArea = true
		}

		// And while we're here, let's build the help text
		areaList += fmt.Sprintf("`%s`, ", k)
	}
	mutex.Unlock()

	// And spiffy up the help message a little...
	areaList = trimSuffix(strings.TrimSpace(areaList), ",")
	helpMsg += areaList

	// If we didn't find the area the user wanted, then show
	// the help message
	if foundArea == false {
		message = helpMsg
	} else {
		// Ah we have something to return to them
		message = areaStatus
	}

	return message
}

func checkForCommands(input string) (bool, string) {
	response := ""
	sendResponse := false
	fmt.Println("Got this:", input)
	matched, _ := regexp.MatchString("!area", input)
	if matched {
		sendResponse = true
		response = reportForArea(strings.TrimSpace(input))
	}

	return sendResponse, response
}

func keepTrackOfAreas() {
	for {
		// Get our message from the MQTT topic
		statusMsg := <-statusChannel

		// And split it up into its various parts
		lineParts := strings.Split(statusMsg.spaceStatus, ",")
		i, err := strconv.ParseInt(lineParts[0], 10, 64)
		if err != nil {
			panic(err)
		}

		// Okay, the message we got from the topic should be in the
		// form of:
		//		1597446363,Lasers-1:CNC Lounge,1
		// Here we're not interested in the individual sensor but the area
		// so going to take lineParts[1] and further split that into what
		// we want
		area := strings.Split(lineParts[1], ":")[1]
		if len(area) == 0 {
			fmt.Println("No area for", lineParts[1])
			continue
		}

		// Convert the unix timestamp to a time object for the map
		tm := time.Unix(i, 0)
		mutex.Lock()

		// Check to see if there's already an entry for this sensor in the map
		if _, testIfExists := sensorMap[area]; testIfExists {
			// There is an entry, but it has an older timestamp
			// so we are going to simply delete it, because we know that
			// the timestamp we have right now is newer than this one (even if
			// by a second), and that's all we care about
			delete(sensorMap, area)
		}
		// Now we add our entry into the map so the buildTimeLine() function
		// can evaulate it
		sensorMap[area] = tm
		mutex.Unlock()
	}
}

func main() {
	fmt.Println("Okay, here we go...")

	// The channel we're going to receive messages on
	statusChannel = make(chan StatusMessage)

	// Create our map that will hold the key of area
	// to its timestamp
	sensorMap = make(map[string]time.Time)

	// Now start the mqtt stuff so we can start getting messages
	go listenOnTopic()

	// And start our bookkeeping routine
	go keepTrackOfAreas()

	//
	// Now begins the Slack stuff
	//
	token := "" // ToDo: Need the appropriate API token from Slack
	api := slack.New(token)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				text := ev.Text
				text = strings.TrimSpace(text)
				text = strings.ToLower(text)

				// Let's see if someone asked us for something...
				sendResponse, response := checkForCommands(text)

				if sendResponse {
					// ...yep, we sent something back, so let's send it to the channel
					rtm.SendMessage(rtm.NewOutgoingMessage(response, ev.Channel))
				}

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				// Nothin' to do
			}
		}
	}
}
