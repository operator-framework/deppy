package universe

import (
	"sort"

	"github.com/operator-framework/deppy/internal/source"
)

type Predicate func(entity *source.Entity) bool

type SearchResult []source.Entity
type SortFunction func(e1 *source.Entity, e2 *source.Entity) bool

func (r SearchResult) Sort(fn SortFunction) SearchResult {
	sort.SliceStable(r, func(i, j int) bool {
		return fn(&r[i], &r[j])
	})
	return r
}

func (r SearchResult) CollectIds() []source.EntityID {
	ids := make([]source.EntityID, len(r))
	for i := range r {
		ids[i] = r[i].ID()
	}
	return ids
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
