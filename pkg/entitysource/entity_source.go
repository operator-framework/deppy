package entitysource

import (
	"context"
)

// IteratorFunction is executed for each entity when iterating over all entities
type IteratorFunction func(entity *Entity) error

// SortFunction returns true if e1 is less than e2
type SortFunction func(e1 *Entity, e2 *Entity) bool

// GroupByFunction transforms an entity into a slice of keys (strings)
// over which the entities will be grouped by
type GroupByFunction func(e1 *Entity) []string

// Predicate returns true if the entity should be kept when filtering
type Predicate func(entity *Entity) bool

type EntityList []Entity
type EntityListMap map[string]EntityList

// EntityQuerier is an interface for querying entities in some store
type EntityQuerier interface {
	Get(ctx context.Context, id EntityID) *Entity
	Filter(ctx context.Context, filter Predicate) (EntityList, error)
	GroupBy(ctx context.Context, fn GroupByFunction) (EntityListMap, error)
	Iterate(ctx context.Context, fn IteratorFunction) error
}

// EntityContentGetter is used to retrieve arbitrary content linked to the
// entities. For instance, the actual package to install, etc.
type EntityContentGetter interface {
	GetContent(ctx context.Context, id EntityID) (interface{}, error)
}

// EntitySource provides a query and content acquisition interface for arbitrary entity stores
type EntitySource interface {
	EntityQuerier
	EntityContentGetter
}

var _ EntitySource = &Group{}

// Group is a simple EntitySource implementation which groups various entity sources
// to provide a single interface by which to query them and get content
type Group struct {
	entitySources []EntitySource
}

func NewGroup(entitySources ...EntitySource) *Group {
	return &Group{
		entitySources: entitySources,
	}
}

// TODO: update implementation to use concurrency, go routines, etc.

func (g *Group) Get(ctx context.Context, id EntityID) *Entity {
	for _, entitySource := range g.entitySources {
		if entity := entitySource.Get(ctx, id); entity != nil {
			return entity
		}
	}
	return nil
}

func (g *Group) Filter(ctx context.Context, filter Predicate) (EntityList, error) {
	resultSet := EntityList{}
	for _, entitySource := range g.entitySources {
		rs, err := entitySource.Filter(ctx, filter)
		if err != nil {
			return nil, err
		}
		resultSet = append(resultSet, rs...)
	}
	return resultSet, nil
}

func (g *Group) GroupBy(ctx context.Context, fn GroupByFunction) (EntityListMap, error) {
	resultSet := EntityListMap{}
	for _, entitySource := range g.entitySources {
		rs, err := entitySource.GroupBy(ctx, fn)
		if err != nil {
			return nil, err
		}
		for key, entities := range rs {
			resultSet[key] = append(resultSet[key], entities...)
		}
	}
	return resultSet, nil
}

func (g *Group) Iterate(ctx context.Context, fn IteratorFunction) error {
	for _, entitySource := range g.entitySources {
		if err := entitySource.Iterate(ctx, fn); err != nil {
			return err
		}
	}
	return nil
}

func (g *Group) GetContent(ctx context.Context, id EntityID) (interface{}, error) {
	for _, entitySource := range g.entitySources {
		if content, err := entitySource.GetContent(ctx, id); err != nil && content != nil {
			return content, err
		}
	}
	return nil, nil
}
