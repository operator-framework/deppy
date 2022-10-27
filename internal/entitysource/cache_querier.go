package entitysource

import (
	"context"

	pkgentitysource "github.com/operator-framework/deppy/pkg/entitysource"
)

var _ pkgentitysource.EntityQuerier = &CacheQuerier{}

type CacheQuerier struct {
	// TODO: separate out a cache
	entities map[pkgentitysource.EntityID]pkgentitysource.Entity
}

func NewCacheQuerier(entities map[pkgentitysource.EntityID]pkgentitysource.Entity) *CacheQuerier {
	return &CacheQuerier{
		entities: entities,
	}
}

func (c CacheQuerier) Get(_ context.Context, id pkgentitysource.EntityID) *pkgentitysource.Entity {
	if entity, ok := c.entities[id]; ok {
		return &entity
	}
	return nil
}

func (c CacheQuerier) Filter(_ context.Context, filter pkgentitysource.Predicate) (pkgentitysource.EntityList, error) {
	resultSet := pkgentitysource.EntityList{}
	for _, entity := range c.entities {
		if filter(&entity) {
			resultSet = append(resultSet, entity)
		}
	}
	return resultSet, nil
}

func (c CacheQuerier) GroupBy(_ context.Context, fn pkgentitysource.GroupByFunction) (pkgentitysource.EntityListMap, error) {
	resultSet := pkgentitysource.EntityListMap{}
	for _, entity := range c.entities {
		keys := fn(&entity)
		for _, key := range keys {
			resultSet[key] = append(resultSet[key], entity)
		}
	}
	return resultSet, nil
}

func (c CacheQuerier) Iterate(_ context.Context, fn pkgentitysource.IteratorFunction) error {
	for _, entity := range c.entities {
		if err := fn(&entity); err != nil {
			return err
		}
	}
	return nil
}
