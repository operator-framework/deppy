package dimacs

import (
	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/entitysource/factory"
)

var _ entitysource.EntitySource = &EntitySource{}

type EntitySource struct {
	entitysource.EntityQuerier
	entitysource.EntityContentGetter
}

func NewDimacsEntitySource(dimacs *Dimacs) *EntitySource {
	entities := make(map[entitysource.EntityID]entitysource.Entity, len(dimacs.Variables()))
	for _, variable := range dimacs.Variables() {
		id := entitysource.EntityID(variable)
		entities[id] = *entitysource.NewEntity(entitysource.EntityID(variable), nil)
	}

	return &EntitySource{
		EntityQuerier:       factory.NewCacheQuerier(entities),
		EntityContentGetter: entitysource.NoContentSource(),
	}
}
