package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/deppy/internal/source"
	"github.com/operator-framework/operator-registry/alpha/property"
	"github.com/operator-framework/operator-registry/pkg/api"
	registry "github.com/operator-framework/operator-registry/pkg/client"
	"google.golang.org/grpc/connectivity"

	"github.com/avast/retry-go/v4"

	"github.com/tidwall/gjson"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ source.DeppySource = &CatalogSourceDeppySource{}

type CatalogSourceDeppySource struct {
	client    client.Client
	name      string
	namespace string

	entities []source.Entity
}

func NewCatalogSourceDeppySource(kubeClient client.Client, namespace string, name string) *CatalogSourceDeppySource {
	return &CatalogSourceDeppySource{
		client:    kubeClient,
		name:      name,
		namespace: namespace,
	}
}

func (c *CatalogSourceDeppySource) Sync(ctx context.Context) error {
	return retry.Do(func() error {
		var catalogSource *v1alpha1.CatalogSource = &v1alpha1.CatalogSource{}
		if err := c.client.Get(ctx, client.ObjectKey{
			Name:      c.name,
			Namespace: c.namespace,
		}, catalogSource); err != nil {
			return err
		}

		if catalogSource.Status.GRPCConnectionState.LastObservedState != connectivity.Ready.String() {
			return fmt.Errorf("catalog source grpc connection is not in a ready state")
		}

		if catalogSource == nil || catalogSource.Status.RegistryServiceStatus == nil {
			return fmt.Errorf("catalog source registry service address undefined")
		}

		registryClient, err := registry.NewClient(catalogSource.Status.RegistryServiceStatus.Address())
		if err != nil {
			return err
		}

		return c.convertCatalogToEntities(ctx, registryClient)
	}, retry.Context(ctx))
}

func (c *CatalogSourceDeppySource) Iterate(_ context.Context, fn source.IteratorFunction) error {
	for _, entity := range c.entities {
		if err := fn(&entity); err != nil {
			return err
		}
	}
	return nil
}

func (c *CatalogSourceDeppySource) Get(ctx context.Context, id source.EntityID) *source.Entity {
	for _, entity := range c.entities {
		if entity.ID() == id {
			return &entity
		}
	}
	return nil
}

func (c *CatalogSourceDeppySource) Filter(_ context.Context, filter source.Predicate) (source.SearchResult, error) {
	resultSet := make([]source.Entity, 0)
	for _, entity := range c.entities {
		if filter(&entity) {
			resultSet = append(resultSet, entity)
		}
	}
	return resultSet, nil
}

func (c *CatalogSourceDeppySource) GroupBy(_ context.Context, fn source.GroupByFunction) (source.GroupByResult, error) {
	resultMap := make(map[string]source.SearchResult)
	for _, entity := range c.entities {
		keys := fn(&entity)
		// open question: do we care about entities whose key(s) is nil
		// i.e. no key could be derived from the entity
		for _, key := range keys {
			resultMap[key] = append(resultMap[key], entity)
		}
	}
	return resultMap, nil
}

func (c *CatalogSourceDeppySource) convertCatalogToEntities(ctx context.Context, client *registry.Client) error {
	bundleIterator, err := client.ListBundles(ctx)
	if err != nil {
		return err
	}

	pkgDefaultChannel := map[string]string{}
	var entities = make([]source.Entity, 0)

	for bundle := bundleIterator.Next(); bundle != nil; bundle = bundleIterator.Next() {
		if _, ok := pkgDefaultChannel[bundle.PackageName]; !ok {
			pkg, err := client.GetPackage(ctx, bundle.PackageName)
			if err != nil {
				return err
			}
			pkgDefaultChannel[bundle.PackageName] = pkg.DefaultChannelName
		}
		deppyEntity, err := c.bundleToDeppyEntity(bundle, pkgDefaultChannel[bundle.PackageName])
		if err != nil {
			return err
		}
		entities = append(entities, *deppyEntity)
	}

	c.entities = entities
	return nil
}

func (c *CatalogSourceDeppySource) bundleToDeppyEntity(apiBundle *api.Bundle, defaultChannel string) (*source.Entity, error) {
	entityId := source.EntityID(fmt.Sprintf("%s:%s:%s:%s:%s", c.namespace, c.name, apiBundle.ChannelName, apiBundle.PackageName, apiBundle.Version))
	properties := map[string]string{}
	for _, prop := range apiBundle.Properties {
		switch prop.Type {
		case property.TypePackage:
			properties["olm.packageName"] = gjson.Get(prop.Value, "packageName").String()
			properties["olm.version"] = gjson.Get(prop.Value, "version").String()
		default:
			if curValue, ok := properties[prop.Type]; ok {
				if curValue[0] != '[' {
					curValue = "[" + curValue + "]"
				}
				properties[prop.Type] = curValue[0:len(curValue)-1] + "," + prop.Value + "]"
			} else {
				properties[prop.Type] = prop.Value
			}
		}
	}
	properties["olm.channel"] = apiBundle.ChannelName
	properties["olm.defaultChannel"] = defaultChannel

	if apiBundle.Replaces != "" {
		properties["olm.replaces"] = apiBundle.Replaces
	}

	if apiBundle.SkipRange != "" {
		properties["olm.skipRange"] = apiBundle.SkipRange
	}

	if len(apiBundle.Skips) > 0 {
		properties["olm.skips"] = fmt.Sprintf("[%s]", strings.Join(apiBundle.Skips, ","))
	}

	return source.NewEntity(entityId, properties), nil
}
