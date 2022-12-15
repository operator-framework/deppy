package input

import (
	"context"

	"github.com/operator-framework/deppy/pkg/solver"
)

var _ EntitySource = &CacheEntitySource{}

type CacheEntitySource struct {
	// TODO: separate out a cache
	entities map[solver.Identifier]Entity
}

func NewCacheQuerier(entities map[solver.Identifier]Entity) *CacheEntitySource {
	return &CacheEntitySource{
		entities: entities,
	}
}

func (c CacheEntitySource) Get(_ context.Context, id solver.Identifier) *Entity {
	if entity, ok := c.entities[id]; ok {
		return &entity
	}
	return nil
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
