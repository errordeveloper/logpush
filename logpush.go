package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"inputs/logfile"
	"log"
	"net/http"
	"outputs/eventsource"
)

type Inputs struct {
	Logfile *logfile.LogfileInput
}

var inputs *Inputs

func ListInputs(rw http.ResponseWriter, req *http.Request) {
	result, err := json.Marshal(inputs)
	if err == nil {
		rw.Write(result)
	} else {
		log.Fatal(err)
	}
}

func main() {
	events := eventsource.NewServer()
	router := mux.NewRouter()

	inputs = &Inputs{
		logfile.Init(),
	}

	router.HandleFunc("/v0/logs/all/realtime", events.ServeHTTP).Methods("GET")

	router.HandleFunc("/v0/logs", ListInputs).Methods("GET")

	go inputs.Logfile.Register("test.log", events.Notifier)

	log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", router))
}
