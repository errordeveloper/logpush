// By all meants based on https://gist.github.com/ismasan/3fb75381cd2deb6bfa9c
package eventsource

import (
	"fmt"
	"log"
	"net/http"
)

type EventSource struct {
	Notifier       chan []byte
	newClients     chan chan []byte
	closingClients chan chan []byte
	clients        map[chan []byte]bool
	// TODO: clients struct {
	//    connected bool
	//    filter_by struct {
	//       // this would provider-dependent and smart providers, such
	//       // as journald, would have quite a few options here, while
	//       // simple once could only have name
	//    }
	// }
}

func InitListener() (eventSource *EventSource) {
	eventSource = &EventSource{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	go eventSource.listen()

	return
}

func (eventSource *EventSource) listen() {
	for {
		select {
		case c := <-eventSource.newClients:
			eventSource.clients[c] = true
			log.Printf("Client added - total clients: %d", len(eventSource.clients))
		case c := <-eventSource.closingClients:
			delete(eventSource.clients, c)
			log.Printf("Client removed - total clients: %d", len(eventSource.clients))
		case e := <-eventSource.Notifier:
			for messages, _ := range eventSource.clients {
				messages <- e
			}
		}
	}
}

func (eventSource *EventSource) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	flusher, ok := rw.(http.Flusher)

	if !ok {
		http.Error(rw, "Streaming unsuported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	messages := make(chan []byte)
	eventSource.newClients <- messages

	defer func() {
		eventSource.closingClients <- messages
	}()

	closeNotify := rw.(http.CloseNotifier).CloseNotify()
	go func() {
		<-closeNotify
		eventSource.closingClients <- messages
	}()

	for {
		fmt.Fprintf(rw, "data: %s\n\n", <-messages)
		flusher.Flush()
	}
}
