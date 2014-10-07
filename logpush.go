package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"inputs/logfile"
	"log"
	"net"
	"net/http"
	"outputs/elasticsearch"
	"outputs/eventsource"
	"strings"
	"time"
)

type Inputs struct {
	BasicLineInput *logfile.BasicLineInput
}

var inputs *Inputs

func ValidateElasticSearchOuput() {
	addr := net.UDPAddr{
		Port: 9700,
		IP:   net.ParseIP("127.0.0.1"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	//defer conn.Close()
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 10000)
	// Do something with `conn`
	for {
		time.Sleep(100 * time.Millisecond)
		go func() {
			// Read the incoming connection into the buffer.
			l, err := conn.Read(buf)
			log.Println("[validator] read", l, "bytes")
			if err != nil {
				log.Fatal("Error reading:", err.Error())
			}

			var f interface{}

			lines := strings.Split(string(buf[:l]), "\n")
			for n := 0; n < (len(lines) - 1); n++ {
				err = json.Unmarshal([]byte(lines[n]), &f)
				if err != nil {
					log.Fatal("Error parsing:", err.Error())
				}
				log.Println(f)
			}
		}()
	}

}

func ListInputs(rw http.ResponseWriter, req *http.Request) {
	result, err := json.Marshal(inputs)
	if err == nil {
		rw.Write(result)
	} else {
		log.Fatal(err)
	}
}

func main() {
	events := eventsource.InitListener()
	es := elasticsearch.InitListener()

	go ValidateElasticSearchOuput()

	router := mux.NewRouter()

	inputs = &Inputs{
		logfile.InitBasicLineInput(),
	}

	router.
		HandleFunc("/v0/logs/all/realtime", events.ServeHTTP).
		Methods("GET")

	router.
		HandleFunc("/v0/logs", ListInputs).
		Methods("GET")

	go inputs.BasicLineInput.Register("test.log",
		events.Notifier,
		es.Notifier,
	)

	log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", router))
}
