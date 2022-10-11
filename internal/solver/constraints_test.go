package solver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrder(t *testing.T) {
	type tc struct {
		Name       string
		Constraint Constraint
		Expected   []Identifier
	}

	for _, tt := range []tc{
		{
			Name:       "mandatory",
			Constraint: Mandatory("a"),
		},
		{
			Name:       "prohibited",
			Constraint: Prohibited("b"),
		},
		{
			Name:       "dependency",
			Constraint: Dependency("a", "b", "c"),
			Expected:   []Identifier{"b", "c"},
		},
		{
			Name:       "conflict",
			Constraint: Conflict("a", "b"),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Expected, tt.Constraint.order())
		})
	}
}
