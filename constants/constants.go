package constants

import "time"

const (
	DefaultResyncPeriod = 1*time.Minute
	// Since events live in the kube-apiserver only for 1 hour, we have to remove
	// old objects to avoid memory leaks. If TTL is exactly 1 hour, race
	// can occur in case of the event being updated right before the end of
	// the hour, since it takes some time to deliver this event via watch.
	// 2 hours ought to be enough for anybody.
	EventStorageTTL = 2 * time.Hour
)