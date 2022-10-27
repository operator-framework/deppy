package solver

import (
	"context"

	"github.com/operator-framework/deppy/pkg/entitysource"
)

// TODO: should be disambiguate between solver errors due to constraints
//       and other generic errors (e.g. entity source not reachable, etc.)

// Solution is returned by the Solver when a solution could be found
// it can be queried by EntityID to see if the entity was selected (true) or not (false)
// by the solver
type Solution map[entitysource.EntityID]bool

type Solver interface {
	Solve(ctx context.Context) (Solution, error)
}
