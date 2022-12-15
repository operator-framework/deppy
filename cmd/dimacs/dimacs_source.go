package dimacs

import (
	"github.com/operator-framework/deppy/pkg/deppy"
	"github.com/operator-framework/deppy/pkg/deppy/input"
)

var _ input.EntitySource = &EntitySource{}

type EntitySource struct {
	*input.CacheEntitySource
}

func NewDimacsEntitySource(dimacs *Dimacs) *EntitySource {
	entities := make(map[deppy.Identifier]input.Entity, len(dimacs.Variables()))
	for _, variable := range dimacs.Variables() {
		id := deppy.Identifier(variable)
		entities[id] = *input.NewEntity(id, nil)
	}

	return &EntitySource{
		CacheEntitySource: input.NewCacheQuerier(entities),
	}
}
