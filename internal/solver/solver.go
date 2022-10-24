package solver

import (
	"context"

	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/sat"
)

var _ Solver = &DeppySolver{}

// Solution is returned by the Solver when a solution could be found
// it can be queried by EntityID to see if the entity was selected (true) or not (false)
// by the solver
type Solution map[entitysource.EntityID]bool

type Solver interface {
	Solve(ctx context.Context) (Solution, error)
}

// DeppySolver is a simple solver implementation that takes an entity source group and a constraint aggregator
// to produce a Solution (or error if no solution can be found)
type DeppySolver struct {
	entitySourceGroup    *entitysource.Group
	constraintAggregator *constraints.ConstraintAggregator
}

func NewDeppySolver(entitySourceGroup *entitysource.Group, constraintAggregator *constraints.ConstraintAggregator) (*DeppySolver, error) {
	return &DeppySolver{
		entitySourceGroup:    entitySourceGroup,
		constraintAggregator: constraintAggregator,
	}, nil
}

func (d DeppySolver) Solve(ctx context.Context) (Solution, error) {
	vars, err := d.constraintAggregator.GetVariables(ctx, d.entitySourceGroup)
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
		if entity := d.entitySourceGroup.Get(ctx, entitysource.EntityID(variable.Identifier())); entity != nil {
			solution[entity.ID()] = false
		}
	}
	for _, variable := range selection {
		if entity := d.entitySourceGroup.Get(ctx, entitysource.EntityID(variable.Identifier())); entity != nil {
			solution[entity.ID()] = true
		}
	}
	return solution, nil
}
