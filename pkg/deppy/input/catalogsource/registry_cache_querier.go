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
	sync.RWMutex
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
		r.logger.Error(err, "error populating initial entity cache")
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
					r.RWMutex.Lock()
					defer r.RWMutex.Unlock()
					delete(r.cache, catalogSourceKey)
					r.logger.Info("Completed cache delete", "catalogSource", catalogSourceKey, "event", entry.Type)
				}()
			case watch.Added, watch.Modified:
				func() {
					r.RWMutex.Lock()
					defer r.RWMutex.Unlock()
					if !catalogSourceReady(catalogSource) {
						return
					}
					imageID, err := r.imageIDFromCatalogSource(ctx, catalogSource)
					if err != nil {
						r.logger.V(1).Info(fmt.Sprintf("failed to get latest imageID for catalogSource %s/%s, skipping cache update: %v", catalogSource.Name, catalogSource.Namespace, err))
						return
					}
					if len(imageID) == 0 {
						// This shouldn't happen, catalogSource would need to be nil
						return
					}
					if oldEntry, ok := r.cache[catalogSourceKey]; ok && imageID == oldEntry.imageID {
						// image hasn't changed since last sync
						return
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
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()

	close(r.done)
}

func (r *CachedRegistryEntitySource) Get(ctx context.Context, id deppy.Identifier) *input.Entity {
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
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
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
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
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
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
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
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
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()
	catalogSourceList := v1alpha1.CatalogSourceList{}
	if err := r.client.List(ctx, &catalogSourceList); err != nil {
		return err
	}
	var errs []error
	for _, catalogSource := range catalogSourceList.Items {
		if !catalogSourceReady(&catalogSource) {
			continue
		}
		imageID, err := r.imageIDFromCatalogSource(ctx, &catalogSource)
		if err != nil {
			errs = append(errs, err)
			continue
		}
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
func (r *CachedRegistryEntitySource) imageIDFromCatalogSource(ctx context.Context, catalogSource *v1alpha1.CatalogSource) (string, error) {
	if catalogSource == nil {
		return "", nil
	}
	svc := corev1.Service{}
	svcName := types.NamespacedName{
		Namespace: catalogSource.Status.RegistryServiceStatus.ServiceNamespace,
		Name:      catalogSource.Status.RegistryServiceStatus.ServiceName,
	}
	if err := r.client.Get(ctx, svcName,
		&svc); err != nil {
		// Cannot detect the backing service, don't refresh cache
		return "", fmt.Errorf("failed to get service %s for catalogSource %s/%s: %v", svcName, catalogSource.Namespace, catalogSource.Name, err)
	}
	if len(svc.Spec.Selector[catalogSourceSelectorLabel]) == 0 {
		// Cannot verify if image is recent
		return "", fmt.Errorf("failed to get pod for service %s for catalogSource %s/%s: missing selector %s", svcName, catalogSource.Namespace, catalogSource.Name, catalogSourceSelectorLabel)
	}
	pods := corev1.PodList{}
	if err := r.client.List(ctx, &pods,
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromValidatedSet(
				map[string]string{catalogSourceSelectorLabel: svc.Spec.Selector[catalogSourceSelectorLabel]},
			)}); err != nil {
		return "", fmt.Errorf("failed to get pod for catalogSource %s/%s: %v", catalogSource.Namespace, catalogSource.Name, err)
	}
	if len(pods.Items) < 1 {
		return "", fmt.Errorf("failed to get pod for catalogSource %s/%s: no pods matching selector %s", catalogSource.Namespace, catalogSource.Name, catalogSourceSelectorLabel)
	}
	if len(pods.Items[0].Status.ContainerStatuses) < 1 {
		// pod not ready
		return "", fmt.Errorf("failed to read imageID from catalogSource pod %s/%s: no containers ready on pod", pods.Items[0].Namespace, pods.Items[0].Name)
	}
	if len(pods.Items[0].Status.ContainerStatuses[0].ImageID) == 0 {
		// pod not ready
		return "", fmt.Errorf("failed to read imageID from catalogSource pod %s/%s: container state: %s", pods.Items[0].Namespace, pods.Items[0].Name, pods.Items[0].Status.ContainerStatuses[0].State)
	}
	return pods.Items[0].Status.ContainerStatuses[0].ImageID, nil
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

func catalogSourceReady(catalogSource *v1alpha1.CatalogSource) bool {
	if catalogSource == nil {
		return false
	}
	if catalogSource.Status.GRPCConnectionState == nil {
		return false
	}
	return catalogSource.Status.GRPCConnectionState.LastObservedState == connectivity.Ready.String()
}
