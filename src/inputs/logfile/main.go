package logfile

import (
  "log"
  "github.com/ActiveState/tail"
)

func Register(filename string, output chan []byte) {
    t, err := tail.TailFile(filename, tail.Config{Follow: true})
    if err == nil {
        //XXX: broker.Notifier <- t.Lines
        for line := range t.Lines {
            output <- []byte(line.Text)
        }
    } else {
        log.Fatal(err)
    }
}
