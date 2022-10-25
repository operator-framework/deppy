package solver

import (
	"context"
	"fmt"

	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/sat"
)

var _ Solver = &DeppySolver{}

type Solution map[entitysource.EntityID]bool

type Solver interface {
	Solve(ctx context.Context) (Solution, error)
}

type Options struct {
	EntitySources     []entitysource.EntitySource
	ConstraintBuilder constraints.ConstraintGenerator
}

type DeppySolver struct {
	entitySources     []entitysource.EntitySource
	constraintBuilder constraints.ConstraintGenerator
}

func NewDeppySolver(opts Options) (*DeppySolver, error) {
	if opts.EntitySources == nil {
		return nil, fmt.Errorf("no entity sources defined for solver")
	}
	if opts.ConstraintBuilder == nil {
		return nil, fmt.Errorf("no constraint generators defined for solver")
	}
	return &DeppySolver{
		entitySources:     opts.EntitySources,
		constraintBuilder: opts.ConstraintBuilder,
	}, nil
}

func (d DeppySolver) Solve(ctx context.Context) (Solution, error) {
	querier := entitysource.NewGroup(d.entitySources...)
	vars, err := d.constraintBuilder.GetVariables(ctx, querier)
	if err != nil {
		return nil, err
	}

	satSolver, err := sat.NewSolver(sat.WithInput(vars))
	if err != nil {
		return nil, err
	}

	selection, err := satSolver.Solve(ctx)
	if err != nil {
		return nil, err
	}

	solution := Solution{}
	for _, variable := range vars {
		if entity := querier.Get(ctx, entitysource.EntityID(variable.Identifier())); entity != nil {
			solution[entity.ID()] = false
		}
	}
	for _, variable := range selection {
		if entity := querier.Get(ctx, entitysource.EntityID(variable.Identifier())); entity != nil {
			solution[entity.ID()] = true
		}
	}
	return solution, nil
}
