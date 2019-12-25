/*
type EventHandler interface {
	OnAdd(event corev1.Event)
	OnUpdate(old, new corev1.Event)
	OnDelete(event corev1.Event)
}
*/
package sinks

import (
	"fmt"

	"github.com/pytimer/kube-events-watcher/events"
	"github.com/pytimer/kube-events-watcher/flags"
)

var defaultMaxBufferSize  = 1000

type Sink interface {
	events.EventHandler

	Run(stopCh <- chan struct{})
}

func NewEventSinkManager(u flags.Uri) (Sink, error){
	switch u.Key {
	case "elasticsearch":
	case "file":
		return newFileSink(u.Val.String())
	}
	return nil, fmt.Errorf("Sink not recognized: %s", u.Key)
}