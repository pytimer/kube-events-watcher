package sinks

import (
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type FileSink struct {
	entryChannel chan *corev1.Event
	writer       io.Writer
	closer       io.Closer
}

func (f FileSink) OnAdd(event *corev1.Event) {
	klog.Info("FileSink OnAdd event, ", event)
	f.entryChannel <- event
}

func (f FileSink) OnUpdate(old, new *corev1.Event) {
	klog.Info("FileSink OnUpdate event, ", old, new)
	f.entryChannel <- new
}

func (f FileSink) OnDelete(event *corev1.Event) {
	klog.Info("FileSink OnDelete event, ", event)
}

func (f FileSink) Run(stopCh <-chan struct{}) {
	for {
		select {
		case entry := <-f.entryChannel:
			_, err := f.writer.Write([]byte(entry.String()+"\n"))
			if err != nil {
				klog.Errorf("File sink write event to file error: %v", err)
				continue
			}
		case <-stopCh:
			klog.Info("File sink recieved stop signal.")
			f.closer.Close()
			return
		}
	}
}

func newFileSink(filename string) (Sink, error) {
	var f *os.File
	var osErr error
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, osErr = os.Create(filename)
	} else {
		f, osErr = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	}

	if osErr != nil {
		return nil, osErr
	}

	return FileSink{
		entryChannel: make(chan *corev1.Event, defaultMaxBufferSize),
		writer:       f,
		closer:       f,
	}, nil
}
