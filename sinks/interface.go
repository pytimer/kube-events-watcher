/*
type EventHandler interface {
	OnAdd(event corev1.Event)
	OnUpdate(old, new corev1.Event)
	OnDelete(event corev1.Event)
}
*/
package sinks

import "github.com/pytimer/kube-events-watcher/events"

type Sink interface {
	events.EventHandler
}
