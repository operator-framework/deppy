package variablesource

import (
	"context"

	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/sat"
)

// VariableSource generates solver constraints given an entity querier interface
type VariableSource interface {
	GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error)
}

var _ VariableSource = &VariableAggregator{}

// VariableAggregator is a simple structure that aggregates different constraint generators
// and collects all generated solver constraints
type VariableAggregator struct {
	constraintGenerators []VariableSource
}

func NewVariableAggregator(constraintGenerators ...VariableSource) *VariableAggregator {
	return &VariableAggregator{
		constraintGenerators: constraintGenerators,
	}
}

func (b *VariableAggregator) GetVariables(ctx context.Context, entityQuerier entitysource.EntityQuerier) ([]sat.Variable, error) {
	// TODO: refactor to scatter cather through go routines
	variables := make([]sat.Variable, 0)
	for _, constraintGenerator := range b.constraintGenerators {
		vars, err := constraintGenerator.GetVariables(ctx, entityQuerier)
		if err != nil {
			return nil, err
		}
		variables = append(variables, vars...)
	}
	return variables, nil
}
