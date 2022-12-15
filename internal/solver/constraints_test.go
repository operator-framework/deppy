package solver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/pkg/deppy/constraint"

	"github.com/operator-framework/deppy/pkg/deppy"
)

func TestOrder(t *testing.T) {
	type tc struct {
		Name       string
		Constraint deppy.Constraint
		Expected   []deppy.Identifier
	}

	for _, tt := range []tc{
		{
			Name:       "mandatory",
			Constraint: constraint.Mandatory(),
		},
		{
			Name:       "prohibited",
			Constraint: constraint.Prohibited(),
		},
		{
			Name:       "dependency",
			Constraint: constraint.Dependency("a", "b", "c"),
			Expected:   []deppy.Identifier{"a", "b", "c"},
		},
		{
			Name:       "conflict",
			Constraint: constraint.Conflict("a"),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Expected, tt.Constraint.Order())
		})
	}
}
