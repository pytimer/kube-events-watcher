package watchers

import (
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// StorageType defines what storage should be used as a cache for the watcher.
type StorageType int

const (
	// SimpleStorage storage type indicates thread-safe map as backing storage.
	SimpleStorage StorageType = iota
	// TTLStorage storage type indicates storage with expiration. When this
	// type of storage is used, TTL should be specified.
	TTLStorage
)

// WatcherStoreConfig represents the configuration of the storage backing the watcher.
type WatcherStoreConfig struct {
	KeyFunc     cache.KeyFunc
	Handler     cache.ResourceEventHandler
	StorageType StorageType
	StorageTTL  time.Duration
}

type watcherStore struct {
	cache.Store
	handler cache.ResourceEventHandler
}

func (s *watcherStore) Add(obj interface{}) error {
	klog.Info("OnAdd event, ", obj)
	return nil
}

func (s *watcherStore) Update(obj interface{}) error {
	klog.Info("OnUpdate event, ", obj)
	return nil
}

func (s *watcherStore) Delete(obj interface{}) error {
	klog.Info("OnDelete event, ", obj)
	return nil
}

func newWatcherStore(config *WatcherStoreConfig) *watcherStore {
	var cacheStorage cache.Store
	switch config.StorageType {
	case TTLStorage:
		cacheStorage = cache.NewTTLStore(config.KeyFunc, config.StorageTTL)
		break
	case SimpleStorage:
	default:
		cacheStorage = cache.NewStore(config.KeyFunc)
		break
	}

	return &watcherStore{
		Store:   cacheStorage,
		handler: config.Handler,
	}
}
