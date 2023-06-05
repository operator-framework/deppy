package cache

import (
	"context"
	"fmt"
	"sync"
)

type Entry[I interface{}] struct {
	Key   Key
	Value I
}

type Updater[I interface{}] interface {
	Start(ctx context.Context) error
	UpdateEventChannel() <-chan UpdateEvent
	EntryStream(sourceKey SourceKey) <-chan Entry[I]
}

type UpdateEventType string
type SourceKey Key

const (
	UpdateEventDelete UpdateEventType = "DELETE"
	UpdateEventUpdate UpdateEventType = "UPDATE"
)

type UpdateEvent struct {
	EventType UpdateEventType
	SourceKey SourceKey
}

func UpdateCacheEvent(sourceKey SourceKey) UpdateEvent {
	return UpdateEvent{
		EventType: UpdateEventUpdate,
		SourceKey: sourceKey,
	}
}

func DeleteCacheEvent(sourceKey SourceKey) UpdateEvent {
	return UpdateEvent{
		EventType: UpdateEventDelete,
		SourceKey: sourceKey,
	}
}

type ManagedPrefixCache[I interface{}] struct {
	cache   *PrefixCache[I]
	updater Updater[I]
	done    chan struct{}
	mu      sync.RWMutex
	cancel  context.CancelFunc
}

func NewManagerCache[I interface{}](updater Updater[I]) *ManagedPrefixCache[I] {
	return &ManagedPrefixCache[I]{
		cache:   NewPrefixCache[I](),
		updater: updater,
		done:    make(chan struct{}),
		mu:      sync.RWMutex{},
		cancel:  nil,
	}
}

func (m *ManagedPrefixCache[I]) Start(parentCtx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		return fmt.Errorf("called Start() on a running managed cache")
	}

	ctx, cancel := context.WithCancel(parentCtx)
	m.cancel = cancel

	if err := m.updater.Start(ctx); err != nil {
		return err
	}

	go func() {
		defer close(m.done)
		for {
			select {
			case <-ctx.Done():
				break
			case event, hasNext := <-m.updater.UpdateEventChannel():
				if !hasNext {
					break
				}
				switch event.EventType {
				case UpdateEventUpdate:
					m.updateCache(event.SourceKey)
				case UpdateEventDelete:
					m.deleteCache(event.SourceKey)
				default:
					break
				}
			}
		}
	}()
	return nil
}

func (m *ManagedPrefixCache[I]) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
}

func (m *ManagedPrefixCache[I]) updateCache(sourceKey SourceKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
}

func (m *ManagedPrefixCache[I]) deleteCache(sourceKey SourceKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache.DeleteByPrefix(Key(sourceKey))
}
