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

var _ ConstraintGenerator = &ConstraintAggregator{}

// ConstraintAggregator is a simple structure that aggregates different constraint generators
// and collects all generated solver constraints
type ConstraintAggregator struct {
	constraintGenerators []ConstraintGenerator
}

func NewConstraintAggregator(constraintGenerators ...ConstraintGenerator) *ConstraintAggregator {
	return &ConstraintAggregator{
		constraintGenerators: constraintGenerators,
	}
}

func (b *ConstraintAggregator) GetVariables(ctx context.Context, entityQuerier entitysource.EntityQuerier) ([]sat.Variable, error) {
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
