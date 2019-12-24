package events

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type EventHandler interface {
	OnAdd(event corev1.Event)
	OnUpdate(old, new corev1.Event)
	OnDelete(event corev1.Event)
}

/*
type ResourceEventHandler interface {
	OnAdd(obj interface{})
	OnUpdate(oldObj, newObj interface{})
	OnDelete(obj interface{})
}
*/
type eventHandlerWrapper struct {
	handler EventHandler
}

func NewEventHandlerWrapper(h EventHandler) *eventHandlerWrapper {
	return &eventHandlerWrapper{handler: h}
}

func (e *eventHandlerWrapper) OnAdd(obj interface{}) {
	klog.Info("OnAdd event, ", obj)

}

func (e *eventHandlerWrapper) OnUpdate(oldObj, newObj interface{}) {
	klog.Info("OnUpdate event, ", oldObj, newObj)
}

func (e *eventHandlerWrapper) OnDelete(obj interface{}) {
	klog.Info("OnDelete event, ", obj)
}
