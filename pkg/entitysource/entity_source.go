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
