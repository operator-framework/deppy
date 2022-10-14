package universe

import (
	"github.com/operator-framework/deppy/internal/source"
)

// TODO: https://github.com/operator-framework/deppy/issues/40

// EntityUniverse represents a queryable set of entities
// it is used as a basis for creating solver constraints
type EntityUniverse struct {
	universe []source.Entity
	idIndex  map[source.EntityID]source.Entity
}

func NewEntityUniverse(entities []source.Entity) *EntityUniverse {
	m := make(map[source.EntityID]source.Entity)
	for _, entity := range entities {
		m[entity.ID()] = entity
	}
	return &EntityUniverse{
		universe: entities,
		idIndex:  m,
	}
}

func (u *EntityUniverse) Get(id source.EntityID) *source.Entity {
	if entity, ok := u.idIndex[id]; ok {
		return &entity
	}
	return nil
}

func (u *EntityUniverse) Search(predicate Predicate) SearchResult {
	out := make([]source.Entity, 0)
	for _, entity := range u.universe {
		if predicate(&entity) {
			out = append(out, entity)
		}
	}
	return out
}

func (u *EntityUniverse) AllEntities() SearchResult {
	return u.universe
}
