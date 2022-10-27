package sat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/pkg/constraints"
)

func TestOrder(t *testing.T) {
	type tc struct {
		Name       string
		Constraint constraints.Constraint
		Expected   []constraints.Identifier
	}

	for _, tt := range []tc{
		{
			Name:       "mandatory",
			Constraint: constraints.Mandatory(),
		},
		{
			Name:       "prohibited",
			Constraint: constraints.Prohibited(),
		},
		{
			Name:       "dependency",
			Constraint: constraints.Dependency("a", "b", "c"),
			Expected:   []constraints.Identifier{"a", "b", "c"},
		},
		{
			Name:       "conflict",
			Constraint: constraints.Conflict("a"),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Expected, tt.Constraint.Order)
		})
	}
}
