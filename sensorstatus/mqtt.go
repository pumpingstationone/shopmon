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
const mqttServer = "tcp://10.10.1.224:1883"

// The topic to listen on. This is specific to how your
// topics are set up on the server. You can listen to more
// than one at a time in a hierarchy with the octothorp ("#")
// character (e.g. "/occupancy/#" )
const topicName = "shopmontopic"

// The clientID must be a unique name for listening on the
// topics, otherwise you may get disconnect errors
const clientID = "sensorstatus"

// This is the topic name that we are going to put our new entry
// on
const webTopicName = "webshopmontopic"
const webClientID = "websensorstatus"

// The client we'll use to publish on
var client MQTT.Client

func onMessageReceived(client MQTT.Client, message MQTT.Message) {
	log.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())
	var sm StatusMessage
	sm.spaceStatus = string(message.Payload())

	// And send it to our channel for processing
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
		log.Printf("Connected to %s to listen\n", mqttServer)
	}

	<-c
}

func setupToPublish() {
	connOpts := MQTT.NewClientOptions().AddBroker(mqttServer).SetClientID(webClientID).SetCleanSession(true)
	client = MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else {
		log.Printf("Connected to %s to publish\n", mqttServer)
	}
}

func publishToTopic(message string) {
	client.Publish(webTopicName, 0, false, message)
}
