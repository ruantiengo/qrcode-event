package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
 	"io"
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
func proxyToGoogle(w http.ResponseWriter, r *http.Request) {
	// Faz uma requisição GET para o Google
	resp, err := http.Get("https://scarneiro.com.br/4pay/sapnow/payment?id=")
	if err != nil {
		http.Error(w, "Erro ao redirecionar para scarneiro", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copia os cabeçalhos da resposta do Google para a resposta do cliente
	for key, values := range resp.Header {
		// Corrige cabeçalhos duplicados ou conflitantes, como "Content-Length"
		if key == "Content-Length" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Define o status da resposta como o status da resposta do Google
	w.WriteHeader(resp.StatusCode)

	// Copia o corpo da resposta do Google para o cliente
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Erro ao copiar resposta: %v", err)
	}
}

func qrCodeHandler(w http.ResponseWriter, r *http.Request) {
	notifyClients("QRCode pago")
	http.Redirect(w, r, "https://scarneiro.com.br/4pay/sapnow/payment?id=", http.StatusFound)
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