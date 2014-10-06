// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// The Initial Developer of the Original Code is the Mozilla Foundation.
// Portions created by the Initial Developer are Copyright (C) 2013-2014
// the Initial Developer. All Rights Reserved.
//
// Contributor(s):
//   Tanguy Leroux (tlrx.dev@gmail.com)
//   Rob Miller (rmiller@mozilla.com)

package elasticsearch

import (
	_ "bytes"
	"errors"
	"fmt"
	_ "io/ioutil"
	"log"
	"net"
	_ "net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ElasticSearchOutput struct {
	// Interval at which accumulated messages should be bulk indexed to
	// ElasticSearch, in milliseconds (default 1000, i.e. 1 second).
	FlushInterval uint32
	// Number of messages that triggers a bulk indexation to ElasticSearch
	// (default to 10)
	FlushCount int
	// ElasticSearch server address. This address also defines the Bulk
	// indexing mode. For example, "http://localhost:9200" defines a server
	// accessible on localhost and the indexing will be done with the HTTP
	// Bulk API, whereas "udp://192.168.1.14:9700" defines a server accessible
	// on the local network and the indexing will be done with the UDP Bulk
	// API. (default to "http://localhost:9200")
	Server string
	// Specify a timeout value in milliseconds for bulk request to complete.
	// Default is 0 (infinite)
	HTTPTimeout uint32
	batchChan   chan []byte
	backChan    chan []byte
	bulkIndexer BulkIndexer
	Notifier    chan []byte
}

func makeIndexPayload(formatedData []byte) []byte {
	preable := fmt.Sprintf(`{"index":{"_index":"logstash-%s","_type":"message"}}`,
		time.Now().UTC().Format("2006-01-02"))

	return []byte(fmt.Sprintf("%s\n%s\n", preable, formatedData))

	/*
		`{
		   "index":{
		     "_index":"logstash-2014.10.05",
		     "_type":"message"
		   }
		 }
		 {
		   "@uuid":"502291ad-3423-40d2-b64e-7f5b8585b9e1",
		   "@timestamp":"2014-10-05T13:10:41.509Z",
		   "@type":"TEST",
		   "@logger":"GoSpec",
		   "@severity":6,
		   "@message":"Test Payload",
		   "@envversion":"0.8",
		   "@pid":63759,
		   "@source_host":"ilya-dmitrichenko.local",
		   "@fields":{
		     "foo":"bar",
		     "number":64
		   }
		 }`
	*/
}

func InitListener() (elasticSearch *ElasticSearchOutput) {
	var err error
	elasticSearch = &ElasticSearchOutput{
		FlushInterval: 5000,
		FlushCount:    5,
		Server:        "udp://localhost:9700",
		HTTPTimeout:   0,
		batchChan:     make(chan []byte),
		backChan:      make(chan []byte, 2),
		Notifier:      make(chan []byte),
	}

	var serverUrl *url.URL
	if serverUrl, err = url.Parse(elasticSearch.Server); err == nil {
		switch strings.ToLower(serverUrl.Scheme) {
		//case "http", "https":
		//	elasticSearch.bulkIndexer = NewHttpBulkIndexer(strings.ToLower(serverUrl.Scheme), serverUrl.Host,
		//		elasticSearch.FlushCount, elasticSearch.HTTPTimeout)
		case "udp":
			elasticSearch.bulkIndexer = NewUDPBulkIndexer(serverUrl.Host, elasticSearch.FlushCount)
		default:
			err = errors.New("Server URL must specify one of `udp`, `http`, or `https`.")
		}
	} else {
		err = fmt.Errorf("Unable to parse ElasticSearch server URL [%s]: %s", elasticSearch.Server, err)
	}

	go elasticSearch.listen()

	return
}

func (elasticSearch *ElasticSearchOutput) listen() {
	var wg sync.WaitGroup
	wg.Add(2)
	go elasticSearch.receiver(&wg)
	go elasticSearch.committer(&wg)
	wg.Wait()
	return
}

// Runs in a separate goroutine, accepting incoming messages, buffering output
// data until the ticker triggers the buffered data should be put onto the
// committer channel.
func (elasticSearch *ElasticSearchOutput) receiver(wg *sync.WaitGroup) {
	var (
		count int
	)
	ok := true
	ticker := time.Tick(time.Duration(elasticSearch.FlushInterval) * time.Millisecond)
	outBatch := make([]byte, 0, 10000)

	for ok {
		select {
		case outBody := <-elasticSearch.Notifier:
			outBytes := makeIndexPayload(outBody)
			outBatch = append(outBatch, outBytes...)
			if count = count + 1; elasticSearch.bulkIndexer.CheckFlush(count, len(outBatch)) {
				if len(outBatch) > 0 {
					// This will block until the other side is ready to accept
					// this batch, so we can't get too far ahead.
					elasticSearch.batchChan <- outBatch
					outBatch = <-elasticSearch.backChan
					log.Println("[buffer] Shipping count", count)
					count = 0
				}
			}
		case <-ticker:
			if len(outBatch) > 0 {
				// This will block until the other side is ready to accept
				// this batch, freeing us to start on the next one.
				elasticSearch.batchChan <- outBatch
				outBatch = <-elasticSearch.backChan
				log.Println("[timer] Shipping count", count)
				count = 0
			}
		}
	}
	wg.Done()
}

// Runs in a separate goroutine, waits for buffered data on the committer
// channel, bulk index it out to the elasticsearch cluster, and puts the now
// empty buffer on the return channel for reuse.
func (elasticSearch *ElasticSearchOutput) committer(wg *sync.WaitGroup) {
	initBatch := make([]byte, 0, 10000)
	elasticSearch.backChan <- initBatch
	var outBatch []byte

	for outBatch = range elasticSearch.batchChan {
		if err := elasticSearch.bulkIndexer.Index(outBatch); err != nil {
			log.Fatal(err)
		}
		outBatch = outBatch[:0]
		elasticSearch.backChan <- outBatch
	}
	wg.Done()
}

// A BulkIndexer is used to index documents in ElasticSearch
type BulkIndexer interface {
	// Index documents
	Index(body []byte) error
	// Check if a flush is needed
	CheckFlush(count int, length int) bool
}

/*
// A HttpBulkIndexer uses the HTTP REST Bulk Api of ElasticSearch
// in order to index documents
type HttpBulkIndexer struct {
	// Protocol (http or https).
	Protocol string
	// Host name and port number (default to "localhost:9200").
	Domain string
	// Maximum number of documents.
	MaxCount int
	// Internal HTTP Client.
	client *http.Client
}

func NewHttpBulkIndexer(protocol string, domain string, maxCount int,
	httpTimeout uint32) *HttpBulkIndexer {

	client := &http.Client{
		Timeout: time.Duration(httpTimeout) * time.Millisecond,
	}
	return &HttpBulkIndexer{
		Protocol: protocol,
		Domain:   domain,
		MaxCount: maxCount,
		client:   client,
	}
}

func (h *HttpBulkIndexer) CheckFlush(count int, length int) bool {
	if count >= h.MaxCount {
		return true
	}
	return false
}

func (h *HttpBulkIndexer) Index(body []byte) error {
	url := fmt.Sprintf("%s://%s%s", h.Protocol, h.Domain, "/_bulk")

	// Creating ElasticSearch Bulk HTTP request
	request, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("can't create bulk request: %s", err.Error())
	}
	request.Header.Add("Accept", "application/json")
	response, err := h.client.Do(request)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %s", err.Error())
	}
	if response != nil {
		defer response.Body.Close()
		if response.StatusCode > 304 {
			return fmt.Errorf("HTTP response error status: %s", response.Status)
		}
		if _, err = ioutil.ReadAll(response.Body); err != nil {
			return fmt.Errorf("can't read HTTP response body: %s", err.Error())
		}
	}
	return nil
}
*/

// A UDPBulkIndexer uses the Bulk UDP Api of ElasticSearch
// in order to index documents
type UDPBulkIndexer struct {
	// Host name and port number (default to "localhost:9700")
	Domain string
	// Maximum number of documents
	MaxCount int
	// Max. length of UDP packets
	MaxLength int
	// Internal UDP Address
	address *net.UDPAddr
	// Internal UDP Client
	client *net.UDPConn
}

func NewUDPBulkIndexer(domain string, maxCount int) *UDPBulkIndexer {
	return &UDPBulkIndexer{Domain: domain, MaxCount: maxCount, MaxLength: 65000}
}

func (u *UDPBulkIndexer) CheckFlush(count int, length int) bool {
	if length >= u.MaxLength {
		return true
	} else if count >= u.MaxCount {
		return true
	}
	return false
}

func (u *UDPBulkIndexer) Index(body []byte) error {
	var err error
	if u.address == nil {
		if u.address, err = net.ResolveUDPAddr("udp", u.Domain); err != nil {
			return fmt.Errorf("Error resolving UDP address [%s]: %s", u.Domain, err)
		}
	}
	if u.client == nil {
		if u.client, err = net.DialUDP("udp", nil, u.address); err != nil {
			return fmt.Errorf("Error creating UDP client: %s", err)
		}
	}
	if u.address != nil {
		if _, err = u.client.Write(body[:]); err != nil {
			return fmt.Errorf("Error writing data to UDP server: %s", err)
		}
	} else {
		return fmt.Errorf("Error writing data to UDP server, address not found")
	}
	return nil
}
