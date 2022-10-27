package constraints

import (
	"context"

	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
	"github.com/operator-framework/deppy/pkg/entitysource"
)

var _ pkgconstraints.ConstraintGenerator = &ConstraintAggregator{}

// ConstraintAggregator is a simple structure that aggregates different constraint generators
// and collects all generated solver constraints
type ConstraintAggregator struct {
	constraintGenerators []pkgconstraints.ConstraintGenerator
}

func NewConstraintAggregator(constraintGenerators []pkgconstraints.ConstraintGenerator) *ConstraintAggregator {
	return &ConstraintAggregator{
		constraintGenerators: constraintGenerators,
	}
}

func (b *ConstraintAggregator) GetVariables(ctx context.Context, entityQuerier entitysource.EntityQuerier) ([]pkgconstraints.IVariable, error) {
	// TODO: refactor to scatter cather through go routines
	variables := make([]pkgconstraints.IVariable, 0)
	for _, constraintGenerator := range b.constraintGenerators {
		vars, err := constraintGenerator.GetVariables(ctx, entityQuerier)
		if err != nil {
			return nil, err
		}
		variables = append(variables, vars...)
	}
	return variables, nil
}
