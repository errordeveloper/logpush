package logfile

import (
  "encoding/json"
  "github.com/ActiveState/tail"
  "log"
  "time"
)

type LogifileMessage struct {
  Name string
  Line string
  TimeRead time.Time
}

func Register(filename string, output chan []byte) {
    t, err := tail.TailFile(filename, tail.Config{Follow: true})
    if err == nil {
        //XXX: broker.Notifier <- t.Lines
        for line := range t.Lines {
            result, _ := json.Marshal(LogifileMessage{filename, line.Text, line.Time})
            output <- result
        }
    } else {
        log.Fatal(err)
    }
}
