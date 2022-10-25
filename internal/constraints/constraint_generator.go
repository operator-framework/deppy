package constraints

import (
	"context"

	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/sat"
)

type ConstraintGenerator interface {
	GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error)
}

var _ ConstraintGenerator = &DeppyConstraintBuilder{}

// DeppyConstraintBuilder concatenates the constraints created by a group of ConstraintGenerators
type DeppyConstraintBuilder struct {
	constraintGenerators []ConstraintGenerator
}

func NewDeppyConstraintBuilder(constraintGenerators ...ConstraintGenerator) *DeppyConstraintBuilder {
	return &DeppyConstraintBuilder{
		constraintGenerators: constraintGenerators,
	}
}

func (b *DeppyConstraintBuilder) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	// TODO: refactor to scatter cather through go routines
	variables := make([]sat.Variable, 0)
	for _, constraintGenerator := range b.constraintGenerators {
		vars, err := constraintGenerator.GetVariables(ctx, querier)
		if err != nil {
			return nil, err
		}
		variables = append(variables, vars...)
	}
	return variables, nil
}
