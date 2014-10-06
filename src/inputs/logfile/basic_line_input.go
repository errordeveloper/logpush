package logfile

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"github.com/ActiveState/tail"
	"log"
	"os"
	"time"
)

type BasicLineInputMessage struct {
	Message    string    `json:"@message"`
	TimeRead   time.Time `json:"@timestamp"`
	Filename   string    `json:"@source_filename"`
	Hostname   string    `json:"@source_hostname"`
	Indetifier string    `json:"@uuid"`
}

type OpenFileInfo struct {
	LinesRead   int64
	ErrorsCount int64
}
type BasicLineInput struct {
	OpenFiles map[string]*OpenFileInfo
}

func (this *BasicLineInput) Register(filename string, output chan []byte) {
	t, err :=
		tail.TailFile(filename, tail.Config{Follow: true})
	hostname, _ := os.Hostname()
	if err == nil {
		this.OpenFiles[filename] = &OpenFileInfo{0, 0}
		for line := range t.Lines {
			this.OpenFiles[filename].LinesRead += 1
			m := BasicLineInputMessage{
				line.Text,
				line.Time,
				filename,
				hostname,
				uuid.NewRandom().String(),
			}
			result, err := json.Marshal(m)
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

func InitBasicLineInput() *BasicLineInput {
	return &BasicLineInput{
		make(map[string]*OpenFileInfo),
	}
}
