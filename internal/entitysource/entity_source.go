package entitysource

import (
	"context"
)

type EntityQuerier interface {
	Get(ctx context.Context, id EntityID) *Entity
	Filter(ctx context.Context, filter Predicate) (SearchResult, error)
	GroupBy(ctx context.Context, fn GroupByFunction) (GroupByResult, error)
	Iterate(ctx context.Context, fn IteratorFunction) error
}

type EntityContentGetter interface {
	GetContent(ctx context.Context, id EntityID) (interface{}, error)
}

type EntitySource interface {
	EntityQuerier
	EntityContentGetter
}

var _ EntitySource = &Group{}

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

func (g *Group) Filter(ctx context.Context, filter Predicate) (SearchResult, error) {
	resultSet := SearchResult{}
	for _, entitySource := range g.entitySources {
		rs, err := entitySource.Filter(ctx, filter)
		if err != nil {
			return nil, err
		}
		resultSet = append(resultSet, rs...)
	}
	return resultSet, nil
}

func (g *Group) GroupBy(ctx context.Context, fn GroupByFunction) (GroupByResult, error) {
	resultSet := GroupByResult{}
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
