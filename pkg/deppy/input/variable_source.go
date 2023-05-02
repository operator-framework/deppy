package input

import (
	"context"

	"github.com/operator-framework/deppy/pkg/deppy"
)

// VariableSource generates solver constraints given an entity querier interface
type VariableSource interface {
	GetVariables(ctx context.Context) ([]deppy.Variable, error)
}

var _ deppy.Variable = &SimpleVariable{}

type SimpleVariable struct {
	id          deppy.Identifier
	constraints []deppy.Constraint
}

func (s *SimpleVariable) Identifier() deppy.Identifier {
	return s.id
}

func (s *SimpleVariable) Constraints() []deppy.Constraint {
	return s.constraints
}

func (s *SimpleVariable) AddConstraint(constraint deppy.Constraint) {
	s.constraints = append(s.constraints, constraint)
}

func NewSimpleVariable(id deppy.Identifier, constraints ...deppy.Constraint) *SimpleVariable {
	return &SimpleVariable{
		id:          id,
		constraints: constraints,
	}
}
