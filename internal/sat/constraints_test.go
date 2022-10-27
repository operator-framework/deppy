package sat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pkgsat "github.com/operator-framework/deppy/pkg/sat"
	"github.com/operator-framework/deppy/pkg/sat/constraints"
)

func TestOrder(t *testing.T) {
	type tc struct {
		Name       string
		Constraint pkgsat.Constraint
		Expected   []pkgsat.Identifier
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
			Expected:   []pkgsat.Identifier{"a", "b", "c"},
		},
		{
			Name:       "conflict",
			Constraint: constraints.Conflict("a"),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Expected, tt.Constraint.Order())
		})
	}
}
