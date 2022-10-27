package constraints

import (
	"context"

	"github.com/operator-framework/deppy/pkg/entitysource"
)

// ConstraintGenerator generates solver constraints given an entity querier interface
type ConstraintGenerator interface {
	GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]IVariable, error)
}
