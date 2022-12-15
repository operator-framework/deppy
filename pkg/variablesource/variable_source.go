package variablesource

import (
	"context"

	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/sat"
)

// VariableSource generates solver constraints given an entity querier interface
type VariableSource interface {
	GetVariables(ctx context.Context, entitySource entitysource.EntitySource) ([]sat.Variable, error)
}
