package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const wwwBind = ":8002"

// The JSON structure which goes over the wire in WebSockets (delimited by newlines)
type wsMessage struct {
	Type  string              `json:"type"`
	Data  map[string]string   `json:"data"`
	AData []map[string]string `json:"adata"`
}

// The memory structure of a single WebSockets client
type wsClient struct {
	ws                 *websocket.Conn
	toClient           chan wsMessage
	fromClient         chan wsMessage
	timeLastFromClient time.Time
	userID             int // 0 = not logged in
	remoteAddr         string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// All currently connected WebSockets clients
var wsClientsLock = WithMutex{}
var wsClients = make(map[*wsClient]time.Time)

// Handles index.html
func wwwHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(200)
	w.Write([]byte("Hello, world!"))
}

// Switches to the WebSockets protocol
func wwwServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	client := wsClient{ws: ws, toClient: make(chan wsMessage, 5), fromClient: make(chan wsMessage, 5), remoteAddr: r.RemoteAddr}
	wsClientsLock.With(func() {
		wsClients[&client] = time.Now()
	})
	go client.handleClient()
}

// goroutine which runs the http + websockets server
func webServer() {
	http.HandleFunc("/", wwwHome)
	http.HandleFunc("/ws", wwwServeWs)
	log.Println("Web server listening on", wwwBind)
	err := http.ListenAndServe(wwwBind, nil)
	if err != nil {
		log.Panic("Cannot listen on ", wwwBind, " for the web server")
	}
}

// Writes a log to the console
func (wsc *wsClient) log(msgs ...interface{}) {
	s := wsc.remoteAddr + ": "
	for i, msg := range msgs {
		if i != 0 {
			s = s + " "
		}
		switch msg.(type) {
		case string:
			s = s + fmt.Sprintf("%s", msg)
		default:
			s = s + fmt.Sprintf("%s", jsonifyWhatever(msg))
		}
	}
	log.Println(s)
}

// goroutine which handles a single websocket client.
func (wsc *wsClient) handleClient() {
	defer func() {
		wsc.ws.Close()
		wsClientsLock.With(func() {
			delete(wsClients, wsc)
		})
	}()
	wsc.log("### Client connected")
	go func() {
		for {
			var msg wsMessage
			err := wsc.ws.ReadJSON(&msg)
			if err != nil {
				msg.Type = "_err"
				msg.Data = map[string]string{"error": err.Error(), "message": "ReadJSON failed"}
				wsc.fromClient <- msg
				return
			}
			wsc.fromClient <- msg
		}
	}()
	for {
		select {
		case msg := <-wsc.toClient:
			// From backend to WS client
			err := wsc.ws.WriteJSON(msg)
			if err != nil {
				wsc.log(err)
				return
			}
		case msg := <-wsc.fromClient:
			// From WS client to backend
			wsc.timeLastFromClient = time.Now()
			wsc.log("--> ", msg)
			switch msg.Type {
			// Handle the special case of communication error
			case "_err":
				wsc.log(msg.Data["error"])
				return
			case "ping":
				wsc.toClient <- wsMessage{Type: "pong", Data: map[string]string{}}
			case "get_status":
				wsc.toClient <- wsMessage{Type: "status", Data: map[string]string{"uptime": fmt.Sprintf("%v", time.Since(startTime))}}
			case "logout":
				wsc.userID = 0
				wsc.toClient <- wsMessage{Type: "logout_ok", Data: map[string]string{}}
			}
		case <-time.After(5 * time.Second):
			if time.Since(wsc.timeLastFromClient) > 120*time.Second {
				wsc.log("Timeout. Last message received at", wsc.timeLastFromClient)
				return
			}
		}
	}
}
