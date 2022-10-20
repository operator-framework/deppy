package entitysource

import (
	"context"

	"github.com/operator-framework/deppy/internal/source"
	"github.com/operator-framework/deppy/internal/universe"
)

type Predicate func(entity *source.Entity) bool
type IteratorFunction func(entity *source.Entity) error
type GroupByFunction func(entity *source.Entity) []string

type EntitySource interface {
	Get(ctx context.Context, id source.EntityID) *source.Entity
	Filter(ctx context.Context, filter Predicate) (universe.EntityUniverse, error)
	GroupBy(ctx context.Context, fn GroupByFunction) (universe.EntityUniverse, error)
	Iterate(ctx context.Context, fn IteratorFunction) error
}

func And(predicates ...Predicate) Predicate {
	return func(entity *source.Entity) bool {
		eval := true
		for _, predicate := range predicates {
			eval = eval && predicate(entity)
			if !eval {
				return false
			}
		}
		return eval
	}
}

func Or(predicates ...Predicate) Predicate {
	return func(entity *source.Entity) bool {
		eval := false
		for _, predicate := range predicates {
			eval = eval || predicate(entity)
			if eval {
				return true
			}
		}
		return eval
	}
}

func Not(predicate Predicate) Predicate {
	return func(entity *source.Entity) bool {
		return !predicate(entity)
	}
}
