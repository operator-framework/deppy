package entitysource

import (
	"context"

	"github.com/operator-framework/deppy/internal/source"
	"github.com/operator-framework/deppy/internal/universe"
)

type Predicate func(entity *source.Entity) bool
type IteratorFunction func(entity *source.Entity) error
type GroupByFunction func(entity *source.Entity) []string

type Sync func(ctx context.Context) (*universe.EntityUniverse, error)

type EntityAdapter interface {
	Sync(ctx context.Context) (*universe.EntityUniverse, error)
}

type EntitySource struct {
	entityUniverse *universe.EntityUniverse
	sync           Sync
}

func NewEntitySource(ctx context.Context, adapter EntityAdapter) (*EntitySource, error) {
	entities, err := adapter.Sync(ctx)
	if err != nil {
		return nil, err
	}

	return &EntitySource{
		entityUniverse: entities,
		sync:           adapter.Sync,
	}, nil
}

func (es *EntitySource) Sync(ctx context.Context) error {
	entities, err := es.sync(ctx)
	if err != nil {
		return err
	}
	es.entityUniverse = entities
	return nil
}

func (es *EntitySource) Get(ctx context.Context, id source.EntityID) *source.Entity {
	for _, entity := range es.entityUniverse.AllEntities() {
		if entity.ID() == id {
			return &entity
		}
	}
	return nil
}
func (es *EntitySource) Filter(ctx context.Context, filter Predicate) (*universe.EntityUniverse, error) {
	resultSet := make([]source.Entity, 0)
	for _, entity := range es.entityUniverse.AllEntities() {
		if filter(&entity) {
			resultSet = append(resultSet, entity)
		}
	}
	return universe.NewEntityUniverse(resultSet), nil
}
func (es *EntitySource) GroupBy(ctx context.Context, fn GroupByFunction) (map[string]universe.EntityUniverse, error) {
	tempMap := make(map[string][]source.Entity)
	for _, entity := range es.entityUniverse.AllEntities() {
		keys := fn(&entity)
		// open question: do we care about entities whose key(s) is nil
		// i.e. no key could be derived from the entity
		for _, key := range keys {
			tempMap[key] = append(tempMap[key], entity)
		}
	}
	resultMap := make(map[string]universe.EntityUniverse)
	for key, val := range tempMap {
		resultMap[key] = *universe.NewEntityUniverse(val)
	}
	return resultMap, nil
}

func (es *EntitySource) Iterate(ctx context.Context, fn IteratorFunction) error {
	for _, entity := range es.entityUniverse.AllEntities() {
		if err := fn(&entity); err != nil {
			return err
		}
	}
	return nil
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
