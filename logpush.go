package main

import (
  "log"
  "net/http"
  "realtime"
  "inputs/logfile"
  "github.com/gorilla/mux"
)

func main() {
    broker := realtime.NewServer()
    router := mux.NewRouter()

    router.HandleFunc("/v0/logs/all/realtime", broker.ServeHTTP)

    go logfile.Register("test.log", broker.Notifier)

    log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", router))
}
