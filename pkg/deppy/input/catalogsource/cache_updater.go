package catalogsource

import (
	"context"
	"fmt"
	"sync"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/deppy/pkg/deppy/input"
	"github.com/operator-framework/deppy/pkg/deppy/input/cache"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ cache.Updater[input.Entity] = &CacheUpdater{}

type SourceReference struct {
	Namespace string
	Name      string
	Address   string
}

type CacheUpdater struct {
	client                  client.WithWatch
	watch                   watch.Interface
	sources                 map[cache.SourceKey]SourceReference
	cacheUpdateEventChannel chan cache.UpdateEvent
	done                    chan struct{}
	mu                      sync.RWMutex
}

func NewCacheUpdater(client client.WithWatch) *CacheUpdater {
	return &CacheUpdater{
		client:                  client,
		mu:                      sync.RWMutex{},
		watch:                   nil,
		cacheUpdateEventChannel: make(chan cache.UpdateEvent),
		sources:                 map[cache.SourceKey]SourceReference{},
		done:                    make(chan struct{}),
	}
}

func (c *CacheUpdater) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.watch != nil {
		return fmt.Errorf("calling CatalogSource CacheUpdater.Start() more than once")
	}
	w, err := c.client.Watch(ctx, &v1alpha1.CatalogSourceList{})
	if err != nil {
		return err
	}
	c.watch = w
	go func() {
		defer close(c.done)
		for {
			select {
			case <-ctx.Done():
				break
			case event, hasNext := <-c.watch.ResultChan():
				if !hasNext {
					break
				}

				// ignore Error and Bookmark events
				if event.Type == watch.Error || event.Type == watch.Bookmark {
					continue
				}

				catalogSource := event.Object.(*v1alpha1.CatalogSource)
				switch event.Type {
				case watch.Modified:
					fallthrough
				case watch.Added:
					if !(isCatalogSourceReady(catalogSource)) {
						continue
					}
					c.handleAddedEvent(catalogSource)
				case watch.Deleted:
					c.handleDeletedEvent(catalogSource)
				default:
					// do nothing - add a log maybe?
				}
			}
		}
	}()
	return nil
}

func (c *CacheUpdater) UpdateEventChannel() <-chan cache.UpdateEvent {
	// TODO: maybe we should multiplex multiple channels?
	return c.cacheUpdateEventChannel
}

func (c *CacheUpdater) Done() <-chan struct{} {
	return c.done
}

func (c *CacheUpdater) EntryStream(sourceKey cache.SourceKey) <-chan cache.Entry[input.Entity] {
	//TODO implement me
	panic("implement me")
}

func (c *CacheUpdater) handleAddedEvent(catalogSource *v1alpha1.CatalogSource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sourceKey := cache.SourceKey(client.ObjectKeyFromObject(catalogSource).String())
	if _, ok := c.sources[sourceKey]; !ok {
		c.sources[sourceKey] = SourceReference{
			Namespace: catalogSource.GetNamespace(),
			Name:      catalogSource.GetName(),
			Address:   catalogSource.Status.GRPCConnectionState.Address,
		}
	}
}

func (c *CacheUpdater) handleDeletedEvent(catalogSource *v1alpha1.CatalogSource) {
	sourceKey := cache.SourceKey(client.ObjectKeyFromObject(catalogSource).String())
	c.safeDeleteSourceReference(sourceKey)
	c.cacheUpdateEventChannel <- cache.DeleteCacheEvent(sourceKey)
}

func (c *CacheUpdater) safeDeleteSourceReference(sourceKey cache.SourceKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.sources, sourceKey)
}

func isCatalogSourceReady(catalogSource *v1alpha1.CatalogSource) bool {
	if catalogSource == nil {
		return false
	}
	if catalogSource.Status.GRPCConnectionState == nil {
		return false
	}
	return catalogSource.Status.GRPCConnectionState.LastObservedState == "READY"
}
