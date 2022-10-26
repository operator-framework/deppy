package entitysource

import "context"

var _ EntityQuerier = &CacheQuerier{}

type CacheQuerier struct {
	// TODO: separate out a cache
	entities map[EntityID]Entity
}

func NewCacheQuerier(entities map[EntityID]Entity) *CacheQuerier {
	return &CacheQuerier{
		entities: entities,
	}
}

func (c CacheQuerier) Get(_ context.Context, id EntityID) *Entity {
	if entity, ok := c.entities[id]; ok {
		return &entity
	}
	return nil
}

func (c CacheQuerier) Filter(_ context.Context, filter Predicate) (EntityList, error) {
	resultSet := EntityList{}
	for _, entity := range c.entities {
		if filter(&entity) {
			resultSet = append(resultSet, entity)
		}
	}
	return resultSet, nil
}

func (c CacheQuerier) GroupBy(_ context.Context, fn GroupByFunction) (EntityListMap, error) {
	resultSet := EntityListMap{}
	for _, entity := range c.entities {
		keys := fn(&entity)
		for _, key := range keys {
			resultSet[key] = append(resultSet[key], entity)
		}
	}
	return resultSet, nil
}

func (c CacheQuerier) Iterate(_ context.Context, fn IteratorFunction) error {
	for _, entity := range c.entities {
		if err := fn(&entity); err != nil {
			return err
		}
	}
	return nil
}
