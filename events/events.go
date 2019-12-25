package events

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

type EventHandler interface {
	OnAdd(event *corev1.Event)
	OnUpdate(old, new *corev1.Event)
	OnDelete(event *corev1.Event)
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
	//klog.Info("OnAdd event, ", obj)
	if event := e.convertToEvent(obj); event != nil {
		e.handler.OnAdd(event)
	}
}

func (e *eventHandlerWrapper) OnUpdate(oldObj, newObj interface{}) {
	//klog.Info("OnUpdate event, ", oldObj, newObj)
	old := e.convertToEvent(oldObj)
	newEvent := e.convertToEvent(newObj)
	if newEvent != nil && old != nil {
		e.handler.OnUpdate(old, newEvent)
	}
}

func (e *eventHandlerWrapper) OnDelete(obj interface{}) {
	//klog.Info("OnDelete event, ", obj)
	event := e.convertToEvent(obj)
	if event == nil {
		// When a delete is dropped, the relist will notice a pod in the store not
		// in the list, leading to the insertion of a tombstone object which contains
		// the deleted key/value. Note that this value might be stale.
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.V(2).Infof("Object is neither event nor tombstone: %+v", obj)
			return
		}
		event, ok = tombstone.Obj.(*corev1.Event)
		if !ok {
			klog.V(2).Infof("Tombstone contains object that is not a pod: %+v", obj)
			return
		}

	}
	e.handler.OnDelete(event)
}

func (e *eventHandlerWrapper) convertToEvent(obj interface{}) *corev1.Event {
	if event, ok := obj.(*corev1.Event); ok {
		return event
	}
	klog.Infof("Watch event handler recevied not event, but %+v", obj)
	return nil
}
