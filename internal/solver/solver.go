package solver

import (
	"context"

	"github.com/operator-framework/deppy/internal/sat"
	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
	"github.com/operator-framework/deppy/pkg/entitysource"
	pkgsolver "github.com/operator-framework/deppy/pkg/solver"
)

var _ pkgsolver.Solver = &DeppySolver{}

// TODO: should be disambiguate between solver errors due to constraints
//       and other generic errors (e.g. entity source not reachable, etc.)

// DeppySolver is a simple solver implementation that takes an entity source group and a constraint aggregator
// to produce a Solution (or error if no solution can be found)
type DeppySolver struct {
	entitySourceGroup    entitysource.EntitySource
	constraintAggregator pkgconstraints.ConstraintGenerator
}

func NewDeppySolver(entitySourceGroup entitysource.EntitySource, constraintAggregator pkgconstraints.ConstraintGenerator) (DeppySolver, error) {
	return DeppySolver{
		entitySourceGroup:    entitySourceGroup,
		constraintAggregator: constraintAggregator,
	}, nil
}

func (d DeppySolver) Solve(ctx context.Context) (pkgsolver.Solution, error) {
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

	solution := pkgsolver.Solution{}
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
