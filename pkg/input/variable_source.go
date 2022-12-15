package input

import (
	"context"

	"github.com/operator-framework/deppy/pkg/solver"
)

// VariableSource generates solver constraints given an entity querier interface
type VariableSource interface {
	GetVariables(ctx context.Context, entitySource EntitySource) ([]solver.Variable, error)
}

var _ solver.Variable = &SimpleVariable{}

type SimpleVariable struct {
	id          solver.Identifier
	constraints []solver.Constraint
}

func (s *SimpleVariable) Identifier() solver.Identifier {
	return s.id
}

func (s *SimpleVariable) Constraints() []solver.Constraint {
	return s.constraints
}

func (s *SimpleVariable) AddConstraint(constraint solver.Constraint) {
	s.constraints = append(s.constraints, constraint)
}

func NewSimpleVariable(id solver.Identifier, constraints ...solver.Constraint) *SimpleVariable {
	return &SimpleVariable{
		id:          id,
		constraints: constraints,
	}
}
