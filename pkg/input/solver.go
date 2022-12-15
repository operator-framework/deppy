package input

import (
	"context"

	"github.com/operator-framework/deppy/pkg/solver"
)

// TODO: should disambiguate between solver errors due to constraints
//       and other generic errors (e.g. entity source not reachable, etc.)

// Solution is returned by the Solver when a solution could be found
// it can be queried by Identifier to see if the entity was selected (true) or not (false)
// by the solver
type Solution map[solver.Identifier]bool

// DeppySolver is a simple solver implementation that takes an entity source group and a constraint aggregator
// to produce a Solution (or error if no solution can be found)
type DeppySolver struct {
	entitySource   EntitySource
	variableSource VariableSource
}

func NewDeppySolver(entitySource EntitySource, variableSource VariableSource) (*DeppySolver, error) {
	return &DeppySolver{
		entitySource:   entitySource,
		variableSource: variableSource,
	}, nil
}

func (d DeppySolver) Solve(ctx context.Context) (Solution, error) {
	vars, err := d.variableSource.GetVariables(ctx, d.entitySource)
	if err != nil {
		return nil, err
	}

	satSolver, err := solver.NewSolver(solver.WithInput(vars))
	if err != nil {
		return nil, err
	}

	selection, err := satSolver.Solve(ctx)
	if err != nil {
		return nil, err
	}

	solution := Solution{}
	for _, variable := range vars {
		if entity := d.entitySource.Get(ctx, variable.Identifier()); entity != nil {
			solution[entity.Identifier()] = false
		}
	}
	for _, variable := range selection {
		if entity := d.entitySource.Get(ctx, variable.Identifier()); entity != nil {
			solution[entity.Identifier()] = true
		}
	}
	return solution, nil
}
