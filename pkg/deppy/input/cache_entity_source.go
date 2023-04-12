package input

import (
	"context"
	"fmt"

	"github.com/operator-framework/deppy/pkg/deppy"
)

var _ EntitySource = &CacheEntitySource{}

type CacheEntitySource struct {
	// TODO: separate out a cache
	entities map[deppy.Identifier]Entity
}

func NewCacheQuerier(entities map[deppy.Identifier]Entity) *CacheEntitySource {
	return &CacheEntitySource{
		entities: entities,
	}
}

func (c CacheEntitySource) Get(_ context.Context, id deppy.Identifier) (*Entity, error) {
	if entity, ok := c.entities[id]; ok {
		return &entity, nil
	}
	return nil, fmt.Errorf("entity with id: %s not found in the entity source", id.String())
}

func (c CacheEntitySource) Filter(_ context.Context, filter Predicate) (EntityList, error) {
	resultSet := EntityList{}
	for _, entity := range c.entities {
		if filter(&entity) {
			resultSet = append(resultSet, entity)
		}
	}
	return resultSet, nil
}

func (c CacheEntitySource) GroupBy(_ context.Context, fn GroupByFunction) (EntityListMap, error) {
	resultSet := EntityListMap{}
	for _, entity := range c.entities {
		keys := fn(&entity)
		for _, key := range keys {
			resultSet[key] = append(resultSet[key], entity)
		}
	}
	return resultSet, nil
}

func (c CacheEntitySource) Iterate(_ context.Context, fn IteratorFunction) error {
	for _, entity := range c.entities {
		if err := fn(&entity); err != nil {
			return err
		}
	}
	return nil
}
