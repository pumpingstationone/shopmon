package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

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

	server := "tcp://glue.pumpingstationone.org:1883" 
	topic := "/occupancy/#"  
	//topic := "/occupancy/hotmetals/1"  
	qos := 0
	clientid := "shopmon"

	connOpts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientid).SetCleanSession(true)

	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(topic, byte(qos), onMessageReceived); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
	}

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else {
		log.Printf("Connected to %s\n", server)
	}

	<-c
}
