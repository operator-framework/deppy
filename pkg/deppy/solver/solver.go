package solver

import (
	"errors"

	"github.com/operator-framework/deppy/internal/solver"
	"github.com/operator-framework/deppy/pkg/deppy"
)

// TODO: should disambiguate between solver errors due to constraints
//       and other generic errors

// Solution is returned by the Solver when the internal solver executed successfully.
// A successful execution of the solver can still end in an error when no solution can
// be found.
type Solution struct {
	err       error
	selection map[deppy.Identifier]deppy.Variable
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

// DeppySolver is a simple solver implementation that takes a slice of variables
// to produce a Solution (or error if no solution can be found)
type DeppySolver struct{}

func NewDeppySolver() *DeppySolver {
	return &DeppySolver{}
}

func (d DeppySolver) Solve(vars []deppy.Variable) (*Solution, error) {
	satSolver, err := solver.New()
	if err != nil {
		return nil, err
	}

	selection, err := satSolver.Solve(vars)
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

	return solution, nil
}
