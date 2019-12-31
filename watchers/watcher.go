package watchers

import (
	"sync"
	"time"

	"github.com/pytimer/kube-events-watcher/constants"
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
		klog.Info("Start watcher")
		e.reflector.Run(stopCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		klog.Info("Start sinks")
		e.sink.Run(stopCh)
	}()

	wg.Wait()
	klog.Info("Stopped")
}

func NewEventWatcher(client kubernetes.Interface, sink sinks.Sink, resyncPeriod time.Duration) Watcher {
	listerWatcher := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (object runtime.Object, e error) {
			klog.Infof("ListFunc %v", options)
			return client.CoreV1().Events(metav1.NamespaceAll).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (i watch.Interface, e error) {
			klog.Infof("WatchFunc %v", options)
			return client.CoreV1().Events(metav1.NamespaceAll).Watch(options)
		},
		DisableChunking: false,
	}

	storeConfig := &WatcherStoreConfig{
		KeyFunc:     cache.DeletionHandlingMetaNamespaceKeyFunc,
		Handler:     events.NewEventHandlerWrapper(sink),
		StorageType: 1,
		StorageTTL:  constants.EventStorageTTL,
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
