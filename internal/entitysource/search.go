package entitysource

import "sort"

type Predicate func(entity *Entity) bool

type EntityList []Entity

func (r EntityList) Sort(fn SortFunction) EntityList {
	sort.SliceStable(r, func(i, j int) bool {
		return fn(&r[i], &r[j])
	})
	return r
}

func (r EntityList) CollectIds() []EntityID {
	ids := make([]EntityID, len(r))
	for i := range r {
		ids[i] = r[i].ID()
	}
	return ids
}

type SortFunction func(e1 *Entity, e2 *Entity) bool

type IteratorFunction func(entity *Entity) error

type GroupByFunction func(e1 *Entity) []string

type GroupedEntityList map[string]EntityList

func (g GroupedEntityList) Sort(fn SortFunction) GroupedEntityList {
	for key, _ := range g {
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
