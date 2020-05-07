package sinks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/estransport"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

var (
	defaultIndexName = "kube-events"
)

type ElasticsearchSink struct {
	client        *elasticsearch.Client
	entryChannel  chan *corev1.Event
	currentBuffer []*corev1.Event
	baseIndex     string
}

func (e *ElasticsearchSink) OnAdd(event *corev1.Event) {
	klog.V(4).Infof("Elasticsearch sink OnAdd event, %v", event)
	e.entryChannel <- event
}

func (e *ElasticsearchSink) OnUpdate(old, new *corev1.Event) {
	klog.V(4).Infof("Elasticsearch sink OnUpdate event, %v", old)
	e.entryChannel <- new
}

func (e *ElasticsearchSink) OnDelete(event *corev1.Event) {
	klog.V(4).Infof("Elasticsearch sink OnDelete event, %v, so skip it.", event)
}

func (e *ElasticsearchSink) Run(stopCh <-chan struct{}) {
	klog.Info("Starting elasticsearch sink...")
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
	klog.V(5).Infof("Ensure elasticsearch sink buffer length: %v", e.currentBuffer)
	go e.sendEntries(entries)
}

func (e *ElasticsearchSink) CreateIndex(name string) error {
	resp, err := e.client.Indices.Exists([]string{name}, e.client.Indices.Exists.WithAllowNoIndices(false))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		klog.V(5).Infof("Ensure the index [%s] already exists, so skip create.", name)
		return nil
	}

	// If index not found, create it.
	if resp.StatusCode == http.StatusNotFound {
		resp, err := e.client.Indices.Create(name)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return fmt.Errorf("create index [%s] error, %s", name, resp)
		}
	}

	return fmt.Errorf("create index [%s] have unknown response code [%d]", name, resp.StatusCode)
}

func (e *ElasticsearchSink) Index(date time.Time) string {
	return date.Format(fmt.Sprintf("%s-2006.01.02", e.baseIndex))
}

func (e *ElasticsearchSink) sendEntries(entries []*corev1.Event) {
	klog.V(1).Infof("Sending %d entries to Elasticsearch", len(entries))

	var buf bytes.Buffer
	for i, entry := range entries {
		indexName := defaultIndexName
		if !entry.EventTime.IsZero() {
			indexName = e.Index(entry.EventTime.Time)
		}
		if entry.EventTime.IsZero() && !entry.FirstTimestamp.IsZero() {
			indexName = e.Index(entry.FirstTimestamp.Time)
		}
		if err := e.CreateIndex(indexName); err != nil {
			klog.Errorf("Failure to create index [%s]: %v", indexName, err)
			return
		}
		// Elasticsearch version less than v6.x, metadata should add "_type": "doc"
		// TODO: generate metadata according to elasticsearch version.
		meta := []byte(fmt.Sprintf(`{"index": {"_index": %s, "_id": "%d", "_type" : "doc"} }%s`, indexName, i+1, "\n"))
		data, err := json.Marshal(entry)
		if err != nil {
			klog.Errorf("Cannot encode event: %v", err)
			continue
		}
		data = append(data, "\n"...)
		buf.Grow(len(meta) + len(data))
		buf.Write(meta)
		buf.Write(data)
	}

	resp, err := e.client.Bulk(bytes.NewReader(buf.Bytes()), e.client.Bulk.WithIndex(defaultIndexName))
	if err != nil {
		klog.Errorf("Failure to send entries to Elasticsearch: %v", err)
		return
	}
	resp.Body.Close()
	buf.Reset()
	klog.V(3).Infof("Successfully sent %d entries to Elasticsearch", len(entries))
}

// sink: elasticsearch:http://<ES_SERVER_URL>?maxRetries=5&debug=true&index=kube-events
func newElasticsearchSink(uri *url.URL) (*ElasticsearchSink, error) {
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url's query string: %s", err)
	}

	address := fmt.Sprintf("%s:%s", uri.Scheme, uri.Host)
	cfg := elasticsearch.Config{
		Addresses: []string{address},
	}

	if len(opts.Get("maxRetries")) > 0 {
		retry, err := strconv.Atoi(opts.Get("maxRetries"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL's maxRetries value into an int: %v", err)
		}
		cfg.MaxRetries = retry
	}

	if len(opts.Get("debug")) > 0 {
		debug, err := strconv.ParseBool(opts.Get("debug"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL's debug value into a bool: %v", err)
		}
		cfg.Logger = &estransport.ColorLogger{Output: os.Stdout, EnableRequestBody: debug}
	}

	index := defaultIndexName
	if len(opts.Get("index")) > 0 {
		index = opts.Get("index")
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ElasticsearchSink{
		client:       esClient,
		entryChannel: make(chan *corev1.Event, defaultMaxBufferSize),
		baseIndex:    index,
	}, nil
}
