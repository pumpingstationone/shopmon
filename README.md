# ShopMon
This is a Go-based website that reads MQTT messages and streams them to a browser using websockets.

The project is based on sensors sending information to an [MQTT](https://mqtt.org) server, and having an application read from that server and do something with the data. In this case, we read the messages from the MQTT server and, depending on whether the sensor indicates there is a someone there, show an animated gif on the map to indicate their presence.

# Design
## Sensors
The sensors used for ShopMon are Raspberry Pi Zero Ws with [this IR sensor](https://www.amazon.com/dp/B012ZZ4LPM/ref=cm_sw_em_r_mt_dp_U_z6XXEbKJ0MWP9). The Pi reads from the sensor continuously and sends a message to the MQTT server with the timestamp and a simple 1 or 0 to indicate the presence of someone, or not, respectively.
 
## Sensor-to-website
The key part of this system is that the sensor knows its name, as that is sent as part of the message to the MQTT server. So, as an example, if ShopMon is going to monitor the wood shop, and there are two wood shop sensors, named `woodshop1` and `woodshop2`, then that name is sent in the message to the MQTT server topic.

ShopMon has a goroutine that is listening on the agreed-upon topic for messages from the sensors. When it reads one from, say, `woodshop1`, it will send that name back to the html page where a CSS id with the same name (e.g. `.woodshop1`) will have the absolute coordinates of where to display the activity gif on the map. 

## Files

### `/img`
This directory contains `ps1firstfloor.jpg`, a blueprint exported from Sketchup from Gary. It also has `activity.gif` which is an animated gif used to indicate whether anyone is occupying a particular space

### `hub.go`
File straight from the [Gorilla](https://github.com/gorilla/websocket) Websocket project for Go

### `mqtt.go`
For listening to messages on the MQTT server on `glue.pumpingstationone.org`. When it sends a message it puts it on an internal channel 

### `main.go`
This file sets up the web server, creates a websocket-based site, and listens on the internal channel for messages from `mqtt.go` above. It streams the messages slightly modified to include a small html snippet to show the `activity.gif` image which is then sent to the html page.

### `shop.html`
The page `main.go` serves up. Uses JavaScript to connect to the server via a websocket, listens for replies and dynamically updates the `<div>`s to show the activity image or not (replaced with `<p/>` in `main.go`).