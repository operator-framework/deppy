package catalogsource

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"google.golang.org/grpc/connectivity"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/deppy/pkg/deppy"
	"github.com/operator-framework/deppy/pkg/deppy/input"
)

const catalogSourceSelectorLabel = "olm.catalogSource"

type CachedRegistryEntitySource struct {
	sync.Mutex
	watch   watch.Interface
	client  client.Client
	rClient RegistryClient
	logger  *logr.Logger
	cache   map[string]sourceCache
	done    chan struct{}
}

type sourceCache struct {
	Items   []*input.Entity
	imageID string
}

func NewCachedRegistryQuerier(watch watch.Interface, client client.Client, rClient RegistryClient, logger *logr.Logger) *CachedRegistryEntitySource {
	return &CachedRegistryEntitySource{
		watch:   watch,
		client:  client,
		rClient: rClient,
		logger:  logger,
		done:    make(chan struct{}),
		cache:   map[string]sourceCache{},
	}
}

func (r *CachedRegistryEntitySource) StartCache(ctx context.Context) {
	// TODO: constraints for limiting watched catalogSources
	// TODO: respect CatalogSource priorities
	if err := r.populate(ctx); err != nil {
		r.logger.Error(err, "failed to populate initial entity cache")
	} else {
		r.logger.Info("Populated initial cache")
	}
	for {
		select {
		case <-r.done:
			return
		case entry := <-r.watch.ResultChan():
			catalogSource, err := catalogSourceFromObject(entry.Object)
			if err != nil {
				r.logger.Error(err, "cannot reconcile catalogSource event", "event", entry.Type, "object", entry.Object)
				continue
			}
			catalogSourceKey := types.NamespacedName{Namespace: catalogSource.Namespace, Name: catalogSource.Name}.String()
			switch entry.Type {
			case watch.Deleted:
				func() {
					r.Mutex.Lock()
					defer r.Mutex.Unlock()
					delete(r.cache, catalogSourceKey)
					r.logger.Info("Completed cache delete", "catalogSource", catalogSourceKey, "event", entry.Type)
				}()
			case watch.Added, watch.Modified:
				func() {
					r.Mutex.Lock()
					defer r.Mutex.Unlock()
					if catalogSource.Status.GRPCConnectionState.LastObservedState != connectivity.Ready.String() {
						return
					}
					imageID := r.imageIDFromCatalogSource(ctx, catalogSource)
					if oldEntry, ok := r.cache[catalogSourceKey]; ok {
						if len(imageID) > 0 && imageID == oldEntry.imageID {
							// image hasn't changed since last sync
							return
						}
					}
					entities, err := r.rClient.ListEntities(ctx, catalogSource)
					if err != nil {
						r.logger.Error(err, "failed to list entities for catalogSource entity cache update", "catalogSource", catalogSourceKey)
						return
					}
					r.cache[catalogSourceKey] = sourceCache{
						Items:   entities,
						imageID: imageID,
					}
					r.logger.Info("Completed cache update", "catalogSource", catalogSourceKey, "event", entry.Type)
				}()
			}
		}
	}
}

func (r *CachedRegistryEntitySource) StopCache() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.done <- struct{}{}
}

func (r *CachedRegistryEntitySource) Get(ctx context.Context, id deppy.Identifier) *input.Entity {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	for _, entries := range r.cache {
		for _, entity := range entries.Items {
			if entity.Identifier() == id {
				return entity
			}
		}
	}
	return nil
}

func (r *CachedRegistryEntitySource) Filter(ctx context.Context, filter input.Predicate) (input.EntityList, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	resultSet := input.EntityList{}
	for _, entries := range r.cache {
		for _, entity := range entries.Items {
			if filter(entity) {
				resultSet = append(resultSet, *entity)
			}
		}
	}
	return resultSet, nil
}

func (r *CachedRegistryEntitySource) GroupBy(ctx context.Context, fn input.GroupByFunction) (input.EntityListMap, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	resultSet := input.EntityListMap{}
	for _, entries := range r.cache {
		for _, entity := range entries.Items {
			keys := fn(entity)
			for _, key := range keys {
				resultSet[key] = append(resultSet[key], *entity)
			}
		}
	}
	return resultSet, nil
}

func (r *CachedRegistryEntitySource) Iterate(ctx context.Context, fn input.IteratorFunction) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	for _, entries := range r.cache {
		for _, entity := range entries.Items {
			if err := fn(entity); err != nil {
				return err
			}
		}
	}
	return nil
}

// populate initializes the state of an empty cache from the catalogSources currently present on the cluster
func (r *CachedRegistryEntitySource) populate(ctx context.Context) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	catalogSourceList := v1alpha1.CatalogSourceList{}
	if err := r.client.List(ctx, &catalogSourceList); err != nil {
		return err
	}
	var errs []error
	for _, catalogSource := range catalogSourceList.Items {
		if catalogSource.Status.GRPCConnectionState.LastObservedState != connectivity.Ready.String() {
			continue
		}
		imageID := r.imageIDFromCatalogSource(ctx, &catalogSource)
		if len(imageID) == 0 {
			continue
		}
		entities, err := r.rClient.ListEntities(ctx, &catalogSource)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		catalogSourceKey := types.NamespacedName{Namespace: catalogSource.Namespace, Name: catalogSource.Name}.String()
		r.cache[catalogSourceKey] = sourceCache{
			Items:   entities,
			imageID: imageID,
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}

// imageIDFromCatalogSource returns the current image ref being run by a catalogSource
func (r *CachedRegistryEntitySource) imageIDFromCatalogSource(ctx context.Context, catalogSource *v1alpha1.CatalogSource) string {
	if catalogSource == nil {
		return ""
	}
	svc := corev1.Service{}
	svcName := types.NamespacedName{
		Namespace: catalogSource.Status.RegistryServiceStatus.ServiceNamespace,
		Name:      catalogSource.Status.RegistryServiceStatus.ServiceName,
	}
	if err := r.client.Get(ctx, svcName,
		&svc); err != nil {
		// Cannot detect the backing service, don't refresh cache
		return ""
	}
	if len(svc.Spec.Selector[catalogSourceSelectorLabel]) == 0 {
		// Cannot verify if image is recent
		return ""
	}
	pods := corev1.PodList{}
	if err := r.client.List(ctx, &pods,
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromValidatedSet(
				map[string]string{catalogSourceSelectorLabel: svc.Spec.Selector[catalogSourceSelectorLabel]},
			)}); err != nil {
		return ""
	}
	if len(pods.Items) < 1 {
		return ""
	}
	if len(pods.Items[0].Status.ContainerStatuses) < 1 {
		// pod not ready
		return ""
	}
	if len(pods.Items[0].Status.ContainerStatuses[0].ImageID) == 0 {
		// pod not ready
		return ""
	}
	return pods.Items[0].Status.ContainerStatuses[0].ImageID
}

func catalogSourceFromObject(obj runtime.Object) (*v1alpha1.CatalogSource, error) {
	// necessary, as obj only accepts direct typecasting to Unstructured
	encodedObj, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("object conversion failed: %v", err)
	}
	catalogSource := v1alpha1.CatalogSource{}
	err = json.Unmarshal(encodedObj, &catalogSource)
	if err != nil {
		return nil, fmt.Errorf("object unmarshalling failed: %v", err)
	}
	return &catalogSource, nil
}
