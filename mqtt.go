package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// MQTTServer is the URL to the MQTT server in the format of
// "tcp://yourservername:port" (port is typically 1883)
const mqttServer = ""

// The topic to listen on. This is specific to how your
// topics are set up on the server. You can listen to more
// than one at a time in a hierarchy with the octothorp ("#")
// character (e.g. "/occupancy/#" )
const topicName = ""

// The clientID must be a unique name for listening on the
// topics, otherwise you may get disconnect errors
const clientID = ""


func onMessageReceived(client MQTT.Client, message MQTT.Message) {
		log.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())
		var sm StatusMessage
		sm.spaceStatus = string(message.Payload())
		// And send it to our buffered channel for the websocket portion to handle
		statusChannel <- sm
}

func listenOnTopic() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	qos := 0

	connOpts := MQTT.NewClientOptions().AddBroker(mqttServer).SetClientID(clientID).SetCleanSession(true)

	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(topicName, byte(qos), onMessageReceived); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
	}

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else {
		log.Printf("Connected to %s\n", mqttServer)
	}

	<-c
}
