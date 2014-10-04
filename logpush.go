package main

import (
  "log"
  "net/http"
  "realtime"
  "inputs/logfile"
)

func main() {
    broker := realtime.NewServer()

    go logfile.Register("test.log", broker.Notifier)

    log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", broker))
}
