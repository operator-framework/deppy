package source

import (
	"context"
)

type EntityQuerier interface {
	Get(ctx context.Context, id EntityID) *Entity
	Filter(ctx context.Context, filter Predicate) (SearchResult, error)
	GroupBy(ctx context.Context, fn GroupByFunction) (GroupByResult, error)
	Iterate(ctx context.Context, fn IteratorFunction) error
}

type DeppySource interface {
	EntityQuerier
}
