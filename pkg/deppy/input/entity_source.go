package input

import (
	"context"
	"sort"

	"github.com/operator-framework/deppy/pkg/deppy"
	"github.com/operator-framework/deppy/pkg/deppy/input/deppyentity"
	"github.com/operator-framework/deppy/pkg/deppy/input/query"
)

// IteratorFunction is executed for each entity when iterating over all entities
type IteratorFunction func(entity *deppyentity.Entity) error

// SortFunction returns true if e1 is less than e2
type SortFunction func(e1 *deppyentity.Entity, e2 *deppyentity.Entity) bool

// GroupByFunction transforms an entity into a slice of keys (strings)
// over which the entities will be grouped by
type GroupByFunction func(e1 *deppyentity.Entity) []string

type EntityList []deppyentity.Entity
type EntityListMap map[string]EntityList

func (r EntityList) Sort(fn SortFunction) EntityList {
	sort.SliceStable(r, func(i, j int) bool {
		return fn(&r[i], &r[j])
	})
	return r
}

func (r EntityList) CollectIds() []deppy.Identifier {
	ids := make([]deppy.Identifier, len(r))
	for i := range r {
		ids[i] = r[i].Identifier()
	}
	return ids
}

func (g EntityListMap) Sort(fn SortFunction) EntityListMap {
	for key := range g {
		sort.SliceStable(g[key], func(i, j int) bool {
			return fn(&g[key][i], &g[key][j])
		})
	}
	return g
}

// EntitySource provides a query and content acquisition interface for arbitrary entity stores
type EntitySource interface {
	Get(ctx context.Context, id deppy.Identifier) *deppyentity.Entity
	Filter(ctx context.Context, filter query.Predicate) (EntityList, error)
	GroupBy(ctx context.Context, fn GroupByFunction) (EntityListMap, error)
	Iterate(ctx context.Context, fn IteratorFunction) error
}
