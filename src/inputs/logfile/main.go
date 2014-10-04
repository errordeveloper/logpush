package logfile

import (
	"encoding/json"
	"github.com/ActiveState/tail"
	"log"
	"time"
)

type LogfileMessage struct {
	Name     string
	Message  string
	TimeRead time.Time
}

type OpenFileInfo struct {
	LinesRead   int64
	ErrorsCount int64
}
type LogfileInput struct {
	OpenFiles map[string]*OpenFileInfo
}

func (this *LogfileInput) Register(filename string, output chan []byte) {
	t, err := tail.TailFile(filename, tail.Config{Follow: true})
	if err == nil {
		this.OpenFiles[filename] = &OpenFileInfo{0, 0}
		//XXX: broker.Notifier <- t.Lines
		for line := range t.Lines {
			this.OpenFiles[filename].LinesRead += 1
			result, err := json.Marshal(LogfileMessage{filename, line.Text, line.Time})
			if err == nil {
				output <- result
			} else {
				log.Fatal(err)
			}
		}
	} else {
		log.Fatal(err)
	}
}

func Init() *LogfileInput {
	return &LogfileInput{
		make(map[string]*OpenFileInfo),
	}
}
