package input

import (
	"sort"

	"github.com/operator-framework/deppy/pkg/deppy"
)

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
func And(predicates ...Predicate) Predicate {
	return func(entity *Entity) bool {
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
	return func(entity *Entity) bool {
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
	return func(entity *Entity) bool {
		return !predicate(entity)
	}
}
