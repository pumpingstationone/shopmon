# shopmon
This is a Go-based website that reads MQTT messages and streams them to a browser using websockets.

## `/img`
This directory contains `ps1firstfloor.jpg`, a blueprint exported from Sketchup from Gary. It also has `activity.gif` which is an animated gif used to indicate whether anyone is occupying a particular space

## `hub.go`
File straight from the [Gorilla](https://github.com/gorilla/websocket) Websocket project for Go

## `mqtt.go`
For listening to messages on the MQTT server on `glue.pumpingstationone.org`. When it sends a message it puts it on an internal channel 

## `main.go`
This file sets up the web server, creates a websocket-based site, and listens on the internal channel for messages from `mqtt.go` above. It streams the messages slightly modified to include a small html snippet to show the `activity.gif` image which is then sent to the html page.

## `shop.html`
The page `main.go` serves up. Uses JavaScript to connect to the server via a websocket, listens for replies and dynamically updates the `<div>`s to show the activity image or not (replaced with `<p/>` in `main.go`).