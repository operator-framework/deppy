package constraints

import (
	"context"

	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/sat"
)

// ConstraintGenerator generates solver constraints given an entity querier interface
type ConstraintGenerator interface {
	GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error)
}
