package main

import (
  _"fmt"
  "github.com/ActiveState/tail"
  "log"
  "net/http"
  "realtime"
)


func main() {
    broker := realtime.NewServer()

    go func() {
        t, err := tail.TailFile("test.log", tail.Config{Follow: true})
        if err == nil {
            //XXX: broker.Notifier <- t.Lines
            for line := range t.Lines {
                broker.Notifier <- []byte(line.Text)
            }
        } else {
            log.Fatal(err)
        }
    }()

    log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", broker))
}
