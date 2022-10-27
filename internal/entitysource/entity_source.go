package entitysource

import (
	"context"

	pkgentitysource "github.com/operator-framework/deppy/pkg/entitysource"
)

var _ pkgentitysource.EntitySource = &Group{}

// Group is a simple EntitySource implementation which groups various entity sources
// to provide a single interface by which to query them and get content
type Group struct {
	entitySources []pkgentitysource.EntitySource
}

func NewGroup(entitySources []pkgentitysource.EntitySource) *Group {
	return &Group{
		entitySources: entitySources,
	}
}

// TODO: update implementation to use concurrency, go routines, etc.

func (g *Group) Get(ctx context.Context, id pkgentitysource.EntityID) *pkgentitysource.Entity {
	for _, entitySource := range g.entitySources {
		if entity := entitySource.Get(ctx, id); entity != nil {
			return entity
		}
	}
	return nil
}

func (g *Group) Filter(ctx context.Context, filter pkgentitysource.Predicate) (pkgentitysource.EntityList, error) {
	resultSet := pkgentitysource.EntityList{}
	for _, entitySource := range g.entitySources {
		rs, err := entitySource.Filter(ctx, filter)
		if err != nil {
			return nil, err
		}
		resultSet = append(resultSet, rs...)
	}
	return resultSet, nil
}

func (g *Group) GroupBy(ctx context.Context, fn pkgentitysource.GroupByFunction) (pkgentitysource.EntityListMap, error) {
	resultSet := pkgentitysource.EntityListMap{}
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

func (g *Group) Iterate(ctx context.Context, fn pkgentitysource.IteratorFunction) error {
	for _, entitySource := range g.entitySources {
		if err := entitySource.Iterate(ctx, fn); err != nil {
			return err
		}
	}
	return nil
}

func (g *Group) GetContent(ctx context.Context, id pkgentitysource.EntityID) (interface{}, error) {
	for _, entitySource := range g.entitySources {
		if content, err := entitySource.GetContent(ctx, id); err != nil && content != nil {
			return content, err
		}
	}
	return nil, nil
}
