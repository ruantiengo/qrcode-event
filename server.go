package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
	"golang.org/x/net/trace"
)

var (
	port = flag.Int("port", 4050, "The server port")
)

type Event struct {
	Message string `json:"message"`
}

var clients = make(map[*websocket.Conn]bool)

func handleWebsocketEchoMessage(ws *websocket.Conn, e Event) error {
	tr := trace.New("websocket.Receive", "receive")
	defer tr.Finish()
	tr.LazyPrintf("Got event %v\n", e)
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
	notifyClients("QRCode pago")

	w.WriteHeader(http.StatusOK)

	http.ServeFile(w, r, "./response.html")
}

func main() {
	flag.Parse()

	http.Handle("/wsecho", websocket.Handler(websocketEchoConnection))
	http.Handle("/wstime", websocket.Handler(websocketTimeConnection))
	http.Handle("/qrcode", http.HandlerFunc(qrCodeHandler))
	http.Handle("/", http.FileServer(http.Dir("static/html")))

	log.Printf("Server listening on port %d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}