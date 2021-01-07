package sinks

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/pytimer/kube-events-watcher/events"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

var (
	defaultIndexName = "kube-events"
)

type ElasticsearchSink struct {
	client        *elastic.Client
	entryChannel  chan *corev1.Event
	currentBuffer []*corev1.Event
	baseIndex     string
	//bulkProcessor *elastic.BulkProcessor
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
	if len(entries) > 0 {
		go e.sendEntries(entries)
	}
}

func (e *ElasticsearchSink) CreateIndex(name string) error {
	ctx := context.Background()
	exists, err := e.client.IndexExists(name).Do(ctx)
	if err != nil {
		return err
	}

	if exists {
		klog.V(5).Infof("Ensure the index [%s] already exists, so skip create.", name)
		return nil
	}

	resp, err := e.client.CreateIndex(name).Do(ctx)
	if err != nil {
		return err
	}
	if !resp.Acknowledged {
		return fmt.Errorf("create index [%s] error, %s", name, err)
	}

	return nil
}

func (e *ElasticsearchSink) Index(date time.Time) string {
	return date.Format(fmt.Sprintf("%s-2006.01.02", e.baseIndex))
}

func (e *ElasticsearchSink) sendEntries(entries []*corev1.Event) {
	klog.V(1).Infof("Sending %d entries to Elasticsearch", len(entries))

	bulkRequest := e.client.Bulk()
	for _, entry := range entries {
		eventEntry := events.Entry{
			Event: entry,
		}
		indexName := defaultIndexName

		if !entry.EventTime.IsZero() {
			eventEntry.Timestamp = entry.EventTime.Time
			indexName = e.Index(entry.EventTime.Time)
		}
		if entry.EventTime.IsZero() && !entry.FirstTimestamp.IsZero() {
			eventEntry.Timestamp = entry.FirstTimestamp.Time
			indexName = e.Index(entry.FirstTimestamp.Time)
		}
		if err := e.CreateIndex(indexName); err != nil {
			klog.Errorf("Failure to create index [%s]: %v", indexName, err)
			return
		}

		bulkRequest.Add(elastic.NewBulkIndexRequest().Index(indexName).Id(string(entry.ObjectMeta.UID)).Doc(eventEntry))
	}

	resp, err := bulkRequest.Do(context.Background())
	if err != nil {
		klog.Warningf("Failure to send entries to Elasticsearch, items: %v, reason: %v", resp.Failed(), err)
		return
	}
	klog.V(3).Infof("Successfully sent %d entries to Elasticsearch", len(entries))
}

// sink: elasticsearch:http://<ES_SERVER_URL>?maxRetries=5&debug=true&index=kube-events&esUserName=&esUserSecret=
func newElasticsearchSink(uri *url.URL) (*ElasticsearchSink, error) {
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url's query string: %s", err)
	}

	var startupFns []elastic.ClientOptionFunc

	address := fmt.Sprintf("%s://%s", uri.Scheme, uri.Host)
	klog.V(3).Infof("Elasticsearch address %s", address)
	startupFns = append(startupFns, elastic.SetURL(address))

	if len(opts.Get("maxRetries")) > 0 {
		retry, err := strconv.Atoi(opts.Get("maxRetries"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL's maxRetries value into an int: %v", err)
		}
		startupFns = append(startupFns, elastic.SetMaxRetries(retry))
	}

	if len(opts.Get("debug")) > 0 {
		debug, err := strconv.ParseBool(opts.Get("debug"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL's debug value into a bool: %v", err)
		}
		if debug {
			startupFns = append(startupFns, elastic.SetInfoLog(wrapKlog{}))
			startupFns = append(startupFns, elastic.SetErrorLog(wrapKlog{}))
			startupFns = append(startupFns, elastic.SetTraceLog(wrapKlog{}))
		}
	}

	index := defaultIndexName
	if len(opts.Get("index")) > 0 {
		index = opts.Get("index")
	}

	if len(opts.Get("esUserName")) > 0 && len(opts.Get("esUserSecret")) > 0 {
		startupFns = append(startupFns, elastic.SetBasicAuth(opts.Get("esUserName"), opts.Get("esUserSecret")))
	}

	esClient, err := elastic.NewClient(startupFns...)
	if err != nil {
		return nil, err
	}

	klog.Info("Elasticsearch sink configuration successfully.")
	return &ElasticsearchSink{
		client:       esClient,
		entryChannel: make(chan *corev1.Event, defaultMaxBufferSize),
		baseIndex:    index,
	}, nil
}

type wrapKlog struct{}

func (l wrapKlog) Printf(format string, v ...interface{}) {
	klog.Infof(format, v...)
}
