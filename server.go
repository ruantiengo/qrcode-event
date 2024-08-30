package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/trace"
	"golang.org/x/net/websocket"
)

var (
	port = flag.Int("port", 4050, "The server port")
)

// Event represents the structure of the messages sent through WebSocket
type Event struct {
	Message string `json:"message"`
}

// Clients map to keep track of connected WebSocket clients
var clients = make(map[*websocket.Conn]bool)

func handleWebsocketEchoMessage(ws *websocket.Conn, e Event) error {
	// Log the request with net.Trace
	tr := trace.New("websocket.Receive", "receive")
	defer tr.Finish()
	tr.LazyPrintf("Got event %v\n", e)

	// Echo the event back as JSON
	err := websocket.JSON.Send(ws, e)
	if err != nil {
		return fmt.Errorf("Can't send: %s", err.Error())
	}
	return nil
}

func websocketEchoConnection(ws *websocket.Conn) {
	log.Printf("Client connected from %s", ws.RemoteAddr())
	clients[ws] = true
	defer func() {
		delete(clients, ws)
		ws.Close()
	}()

	for {
		var event Event
		err := websocket.JSON.Receive(ws, &event)
		if err != nil {
			log.Printf("Receive failed: %s; closing connection...", err.Error())
			break
		} else {
			if err := handleWebsocketEchoMessage(ws, event); err != nil {
				log.Println(err.Error())
				break
			}
		}
	}
}

func websocketTimeConnection(ws *websocket.Conn) {
	for range time.Tick(1 * time.Second) {
		websocket.Message.Send(ws, time.Now().Format("Mon, 02 Jan 2006 15:04:05 PST"))
	}
}

func notifyClients(message string) {
	for client := range clients {
		event := Event{Message: message}
		if err := websocket.JSON.Send(client, event); err != nil {
			log.Printf("Error sending message to client: %s", err.Error())
			client.Close()
			delete(clients, client)
		}
	}
}

func qrCodeHandler(w http.ResponseWriter, r *http.Request) {
	// Notify all connected WebSocket clients about the QR code payment
	notifyClients("QRCode pago")

	// Respond to the HTTP request
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "QR Code payment notification sent to WebSocket clients")
}

func main() {
	flag.Parse()
	// Set up WebSocket servers and static file server. In addition, we're using
	// net/trace for debugging - it will be available at /debug/requests.
	http.Handle("/wsecho", websocket.Handler(websocketEchoConnection))
	http.Handle("/wstime", websocket.Handler(websocketTimeConnection))
	http.Handle("/qrcode", http.HandlerFunc(qrCodeHandler))
	http.Handle("/", http.FileServer(http.Dir("static/html")))

	log.Printf("Server listening on port %d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
