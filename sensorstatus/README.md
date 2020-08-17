# SensorStatus
## Purpose
This is a Go-based intermediary program that converts the sensor data which only indicates when it's on to a feed on a different topic to indicate both on as well as off. 

This program was primarily written for the website, which needs to know when to show both the indicator that the area is occupied, but also needs to know when the area is _not_ occupied so that it can remove the indicator. 

## Design
The way this is done is by reading the sensor topic off the MQTT server, then placing the events in a map, using as a key the sensor name and the timestamp as its value. Go maps allow for duplicate keys but here we replace the existing entry with the same key but the updated timestamp. 

A goroutine `buildTimeLine()` reads the map and if any of the timestamps are older than `expiry` (in seconds), it removes the entry from the map and creates a similar message as what was read off the topic, but with `0` or `1` appended to it, indicating the area is empty or occupied, respectively.

`sendFullStatusMessage()` reads the message off the channel that was popiulated by `buildtimeLine()` and then sends it to a different MQTT topic. 