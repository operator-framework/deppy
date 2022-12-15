package constraints

import "github.com/operator-framework/deppy/pkg/sat"

var _ sat.Variable = &Variable{}

// Variable is a simple implementation of sat.Variable
type Variable struct {
	id          sat.Identifier
	constraints []sat.Constraint
}

func (v *Variable) Identifier() sat.Identifier {
	return v.id
}

func (v *Variable) Constraints() []sat.Constraint {
	return v.constraints
}

func (v *Variable) AddConstraint(constraint ...sat.Constraint) {
	v.constraints = append(v.constraints, constraint...)
}

func NewVariable(id sat.Identifier, constraints ...sat.Constraint) *Variable {
	return &Variable{
		id:          id,
		constraints: constraints,
	}
}
