//package elasticsearch
package main

import (
	"code.google.com/p/go-uuid/uuid"
	. "github.com/errordeveloper/heka/message"
	es "github.com/errordeveloper/heka/plugins/elasticsearch"
	"os"
	"time"
)

func getTestMessage() *Message {
	hostname, _ := os.Hostname()
	field, _ := NewField("foo", "bar", "")
	field1, _ := NewField("number", 64, "")
	msg := &Message{}
	msg.SetType("TEST")
	msg.SetTimestamp(time.Now().UnixNano())
	msg.SetUuid(uuid.NewRandom())
	msg.SetLogger("GoSpec")
	msg.SetSeverity(int32(6))
	msg.SetPayload("Test Payload")
	msg.SetEnvVersion("0.8")
	msg.SetPid(int32(os.Getpid()))
	msg.SetHostname(hostname)
	msg.AddField(field)
	msg.AddField(field1)

	return msg
}

func main() {
	encoder := new(es.ESLogstashV0Encoder)
	config := encoder.ConfigStruct()
	config.(*es.ESLogstashV0EncoderConfig).RawBytesFields = []string{
		"test_raw_field_string",
		"test_raw_field_bytes",
		"test_raw_field_string_array",
		"test_raw_field_bytes_array",
	}

	_ = encoder.Init(config)
	b, _ := encoder.EncodeMessage(getTestMessage())

	x := es.NewUDPBulkIndexer("localhost:9700", 1200)
	for {
		x.Index(b)
	}
}
