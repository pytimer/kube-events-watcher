package sinks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/estransport"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type ElasticsearchSink struct {
	client        *elasticsearch.Client
	entryChannel  chan *corev1.Event
	currentBuffer []*corev1.Event
}

func (e *ElasticsearchSink) OnAdd(event *corev1.Event) {
	e.entryChannel <- event
}

func (e *ElasticsearchSink) OnUpdate(old, new *corev1.Event) {
	e.entryChannel <- new
}

func (e *ElasticsearchSink) OnDelete(event *corev1.Event) {
}

func (e *ElasticsearchSink) Run(stopCh <-chan struct{}) {
	t := time.NewTicker(5 * time.Second)
	for {
		select {
		case entry := <-e.entryChannel:
			if e.currentBuffer == nil {
				klog.V(4).Info("Elasticsearch sink current buffer nil")
				e.currentBuffer = make([]*corev1.Event, 0)
			}
			e.currentBuffer = append(e.currentBuffer, entry)
			if len(e.currentBuffer) >= defaultMaxBufferSize {
				go e.flush()
			}
		case <-t.C:
			go e.flush()
		case <-stopCh:
			klog.Info("Elasticsearch sink recieved stop signal.")
			t.Stop()
			e.flush()
			return
		}
	}
}

func (e *ElasticsearchSink) flush() {
	entries := e.currentBuffer
	e.currentBuffer = nil
	klog.V(4).Infof("Elasticsearch sink flush buffer: %v", e.currentBuffer)
	go e.sendEntries(entries)
}

func (e *ElasticsearchSink) sendEntries(entries []*corev1.Event) {
	klog.V(3).Infof("Sending %d entries to Elasticsearch", len(entries))

	var buf bytes.Buffer
	for i, entry := range entries {
		// Elasticsearch version less than v6.x, metadata should add "_type": "doc"
		// TODO: generate metadata according to elasticsearch version.
		meta := []byte(fmt.Sprintf(`{"index": {"_id": "%d", "_type" : "doc"} }%s`, i+1, "\n"))
		data, err := json.Marshal(entry)
		if err != nil {
			klog.Errorf("Cannot encode event: %v", err)
			continue
		}
		data = append(data, "\n"...)
		buf.Grow(len(meta)+len(data))
		buf.Write(meta)
		buf.Write(data)
	}

	resp, err := e.client.Bulk(bytes.NewReader(buf.Bytes()), e.client.Bulk.WithIndex("kube-events"))
	if err != nil {
		klog.Errorf("Failure to send entries to Elasticsearch: %v", err)
		return
	}
	resp.Body.Close()
	buf.Reset()
	klog.V(3).Infof("Successfully sent %d entries to Elasticsearch", len(entries))
}

func newElasticsearchSink(uri string) (*ElasticsearchSink, error) {
	cfg := elasticsearch.Config{
		Addresses:  []string{uri},
		MaxRetries: 5,
		Logger:     &estransport.ColorLogger{Output: os.Stdout, EnableRequestBody:true},
	}
	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ElasticsearchSink{
		client:       esClient,
		entryChannel: make(chan *corev1.Event, defaultMaxBufferSize),
	}, nil
}
