package solver

import (
	"github.com/operator-framework/deppy/pkg/deppy"
)

// zeroVariable is returned by VariableOf in error cases.
type zeroVariable struct{}

var _ deppy.Variable = zeroVariable{}

func (zeroVariable) Identifier() deppy.Identifier {
	return ""
}

func (zeroVariable) Constraints() []deppy.Constraint {
	return nil
}
