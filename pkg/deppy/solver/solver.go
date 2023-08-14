package solver

import (
	"context"
	"errors"

	"github.com/operator-framework/deppy/internal/solver"
	"github.com/operator-framework/deppy/pkg/deppy"
	"github.com/operator-framework/deppy/pkg/deppy/input"
)

// TODO: should disambiguate between solver errors due to constraints
//       and other generic errors (e.g. entity source not reachable, etc.)

// Solution is returned by the Solver when the internal solver executed successfully.
// A successful execution of the solver can still end in an error when no solution can
// be found.
type Solution struct {
	err       error
	selection map[deppy.Identifier]deppy.Variable
	variables []deppy.Variable
}

// Error returns the resolution error in case the problem is unsat
// on successful resolution, it will return nil
func (s *Solution) Error() error {
	return s.err
}

// SelectedVariables returns the variables that were selected by the solver
// as part of the solution
func (s *Solution) SelectedVariables() map[deppy.Identifier]deppy.Variable {
	return s.selection
}

// IsSelected returns true if the variable identified by the identifier was selected
// in the solution by the resolver. It will return false otherwise.
func (s *Solution) IsSelected(identifier deppy.Identifier) bool {
	_, ok := s.selection[identifier]
	return ok
}

// AllVariables returns all the variables that were considered by the solver to obtain (or not)
// a solution. Note: This is only be present if the AddAllVariablesToSolution option is passed in to the
// Solve call that generated the solution.
func (s *Solution) AllVariables() []deppy.Variable {
	return s.variables
}

type solutionOptions struct {
	addVariablesToSolution bool
}

func (s *solutionOptions) apply(options ...Option) *solutionOptions {
	for _, applyOption := range options {
		applyOption(s)
	}
	return s
}

func defaultSolutionOptions() *solutionOptions {
	return &solutionOptions{
		addVariablesToSolution: false,
	}
}

type Option func(solutionOptions *solutionOptions)

// AddAllVariablesToSolution is a Solve option that instructs the solver to include
// all the variables considered to the Solution it produces
func AddAllVariablesToSolution() Option {
	return func(solutionOptions *solutionOptions) {
		solutionOptions.addVariablesToSolution = true
	}
}

// DeppySolver is a simple solver implementation that takes an entity source group and a constraint aggregator
// to produce a Solution (or error if no solution can be found)
type DeppySolver struct {
	variableSource input.VariableSource
}

func NewDeppySolver(variableSource input.VariableSource) *DeppySolver {
	return &DeppySolver{
		variableSource: variableSource,
	}
}

func (d DeppySolver) Solve(ctx context.Context, options ...Option) (*Solution, error) {
	solutionOpts := defaultSolutionOptions().apply(options...)

	vars, err := d.variableSource.GetVariables(ctx)
	if err != nil {
		return nil, err
	}

	satSolver, err := solver.NewSolver(solver.WithInput(vars))
	if err != nil {
		return nil, err
	}

	selection, err := satSolver.Solve(ctx)
	if err != nil && !errors.As(err, &deppy.NotSatisfiable{}) {
		return nil, err
	}

	selectionMap := map[deppy.Identifier]deppy.Variable{}
	for _, variable := range selection {
		selectionMap[variable.Identifier()] = variable
	}

	solution := &Solution{selection: selectionMap}
	if err != nil {
		unsatError := deppy.NotSatisfiable{}
		errors.As(err, &unsatError)
		solution.err = unsatError
	}

	if solutionOpts.addVariablesToSolution {
		solution.variables = vars
	}

	return solution, nil
}
