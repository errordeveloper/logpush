package main

import (
  "fmt"
  "github.com/ActiveState/tail"
  "log"
  "net/http"
)

type Broker struct {
  Notifier chan []byte
  newClients chan chan []byte
  closingClients chan chan []byte
  clients map[chan []byte]bool
}

func NewServer() (broker *Broker) {
    broker = &Broker{
      Notifier:       make(chan []byte, 1),
      newClients:     make(chan chan []byte),
      closingClients: make(chan chan []byte),
      clients:        make(map[chan []byte]bool),
    }
    go broker.listen()

    return
}

func (broker *Broker) listen() {
    for {
        select {
            case c := <- broker.newClients:
                broker.clients[c] = true
                log.Printf("Client added - total clients: %d", len(broker.clients))
            case c := <- broker.closingClients:
                delete(broker.clients, c)
                log.Printf("Client removed - total clients: %d", len(broker.clients))
            case e := <- broker.Notifier:
                for messages, _ := range broker.clients {
                    messages <- e
                }
        }
    }
}

func (broker *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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
    broker.newClients <- messages

    defer func() {
        broker.closingClients <- messages
    }()

    closeNotify := rw.(http.CloseNotifier).CloseNotify()
    go func() {
        <- closeNotify
        broker.closingClients <- messages
    }()

    for {
        fmt.Fprintf(rw, "data: %s\n\n", <-messages)
        flusher.Flush()
    }
}

func main() {
    broker := NewServer()

    go func() {
        t, err := tail.TailFile("test.log", tail.Config{Follow: true})
        if err == nil {
            //XXX: broker.Notifier <- t.Lines
            for line := range t.Lines {
                broker.Notifier <- []byte(line.Text)
            }
        } else {
            panic(err)
        }
    }()

    log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", broker))

}
