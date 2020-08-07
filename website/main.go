package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// StatusMessage is a struct that is passed from the
// MQTT goroutine (see listenOnTopic() in mqtt.go)
// to the readPump() function in this file. It is
// a struct so that more fields can be added later
// if necessary
type StatusMessage struct {
	spaceStatus string
}

// Our channel that accepts StatusMessages
var statusChannel chan StatusMessage

var addr = flag.String("addr", ":8080", "http service address")

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		statusMsg := <-statusChannel

		// We get a message that is in the form of:
		// 		timestamp,sensor
		//
		// By virtue of the fact we got a message means that
		// we should indicate someone is in the space
		// What we are going to do is send back to the client
		// the message with an html snippet appended to it.
		//
		// Note that the CSS on the webpage sets up the size of the
		// image via the "pulse" class, and it will use the topic
		// name for matching against the css id.
		statusHTML := "<img class=\"pulse\" src=\"/img/activity.gif\"/>"

		// And piece it all together
		msgToSend := fmt.Sprintf("%s:%s", statusMsg.spaceStatus, statusHTML)

		// And send t back to the client over the socket
		log.Printf("Sending %s\n", msgToSend)
		c.hub.broadcast <- []byte(msgToSend)

		// To prevent overwhelming the client and to give the status
		// a couple of seconds to actually show
		time.Sleep(2 * time.Second)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

// Our standard webserver handler
func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.ServeFile(w, r, "shop.html")
}

func main() {

	// Create a buffered channel; 200 is arbitrary but figured based
	// on the number of sensors + the time it takes to fully come up to
	// speed
	statusChannel = make(chan StatusMessage, 200)

	// Set us up to listen to the topics on the MQTT server...
	go listenOnTopic()

	// From here on out we're setting up the websockets layer
	// and spinning up the webserver
	flag.Parse()

	// And set up our hub and run it
	hub := newHub()
	go hub.run()

	// Serve the images from the "img" directory
	fileServer := http.FileServer(http.Dir("./img/"))
	http.Handle("/img/", http.StripPrefix("/img", fileServer))

	// Set up the URLs we can handle
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// And here we go!
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	fmt.Println("hello world")
}
