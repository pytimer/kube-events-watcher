package watchers

import (
	"sync"
	"time"

	"github.com/pytimer/kube-events-watcher/events"
	"github.com/pytimer/kube-events-watcher/sinks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	// Since events live in the kube-apiserver only for 1 hour, we have to remove
	// old objects to avoid memory leaks. If TTL is exactly 1 hour, race
	// can occur in case of the event being updated right before the end of
	// the hour, since it takes some time to deliver this event via watch.
	// 2 hours ought to be enough for anybody.
	eventStorageTTL = 2 * time.Hour
)

type Watcher interface {
	Run(ch <-chan struct{})
}

type eventWatcher struct {
	reflector *cache.Reflector
	sink      sinks.Sink
}

func (e *eventWatcher) Run(stopCh <-chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		klog.Info("start watcher")
		e.reflector.Run(stopCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		klog.Info("start sinks")
		e.sink.Run(stopCh)
	}()

	klog.Info("stopped")
	wg.Wait()
}

func NewEventWatcher(client kubernetes.Interface, sink sinks.Sink, resyncPeriod time.Duration) Watcher {
	listerWatcher := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (object runtime.Object, e error) {
			klog.Info("ListFunc %v", options)
			list, err := client.CoreV1().Events(metav1.NamespaceAll).List(options)
			if err != nil {

			}
			return list, nil
		},
		WatchFunc: func(options metav1.ListOptions) (i watch.Interface, e error) {
			klog.Info("WatchFunc %v", options)
			return client.CoreV1().Events(metav1.NamespaceAll).Watch(options)
		},
		DisableChunking: false,
	}

	storeConfig := &WatcherStoreConfig{
		KeyFunc:     cache.DeletionHandlingMetaNamespaceKeyFunc,
		Handler:     events.NewEventHandlerWrapper(sink),
		StorageType: 1,
		StorageTTL:  eventStorageTTL,
	}

	return &eventWatcher{
		reflector: cache.NewReflector(
			listerWatcher,
			&corev1.Event{},
			newWatcherStore(storeConfig),
			resyncPeriod,
		),
		sink: sink,
	}
}
