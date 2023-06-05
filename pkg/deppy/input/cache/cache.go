package cache

import "sync"

type Key string

type Cache[I interface{}] interface {
	Get(key Key) (I, bool)
	Set(key Key, value I)
	Delete(key Key)
	Iterate(func(key Key, value I) error) error
}

var _ Cache[interface{}] = &MapCache[interface{}]{}

type MapCache[I interface{}] struct {
	cache map[Key]I
	mu    sync.RWMutex
}

func NewMapCache[I interface{}]() *MapCache[I] {
	return &MapCache[I]{
		cache: map[Key]I{},
		mu:    sync.RWMutex{},
	}
}

func (m *MapCache[I]) Get(key Key) (I, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.cache[key]
	return value, ok
}

func (m *MapCache[I]) Set(key Key, value I) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = value
}

func (m *MapCache[I]) Delete(key Key) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
}

func (m *MapCache[I]) Iterate(fn func(key Key, value I) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for key, value := range m.cache {
		if err := fn(key, value); err != nil {
			return err
		}
	}
	return nil
}
