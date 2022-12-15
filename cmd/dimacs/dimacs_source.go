package dimacs

import (
	"github.com/operator-framework/deppy/pkg/input"
	"github.com/operator-framework/deppy/pkg/solver"
)

var _ input.EntitySource = &EntitySource{}

type EntitySource struct {
	*input.CacheEntitySource
}

func NewDimacsEntitySource(dimacs *Dimacs) *EntitySource {
	entities := make(map[solver.Identifier]input.Entity, len(dimacs.Variables()))
	for _, variable := range dimacs.Variables() {
		id := solver.Identifier(variable)
		entities[id] = *input.NewEntity(id, nil)
	}

	return &EntitySource{
		CacheEntitySource: input.NewCacheQuerier(entities),
	}
}
