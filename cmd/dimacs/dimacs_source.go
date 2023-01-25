package dimacs

import (
	"github.com/operator-framework/deppy/pkg/deppy"
	"github.com/operator-framework/deppy/pkg/deppy/input"
	"github.com/operator-framework/deppy/pkg/deppy/input/deppyentity"
)

var _ input.EntitySource = &EntitySource{}

type EntitySource struct {
	*input.CacheEntitySource
}

func NewDimacsEntitySource(dimacs *Dimacs) *EntitySource {
	entities := make(map[deppy.Identifier]deppyentity.Entity, len(dimacs.Variables()))
	for _, variable := range dimacs.Variables() {
		id := deppy.Identifier(variable)
		entities[id] = *deppyentity.NewEntity(id, nil)
	}

	return &EntitySource{
		CacheEntitySource: input.NewCacheQuerier(entities),
	}
}
