package solver

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestVariable struct {
	identifier  Identifier
	constraints []Constraint
}

func (i TestVariable) Identifier() Identifier {
	return i.identifier
}

func (i TestVariable) Constraints() []Constraint {
	return i.constraints
}

func (i TestVariable) GoString() string {
	return fmt.Sprintf("%q", i.Identifier())
}

func variable(id Identifier, constraints ...Constraint) Variable {
	return TestVariable{
		identifier:  id,
		constraints: constraints,
	}
}

func TestNotSatisfiableError(t *testing.T) {
	type tc struct {
		Name   string
		Error  NotSatisfiable
		String string
	}

	for _, tt := range []tc{
		{
			Name:   "nil",
			String: "constraints not satisfiable",
		},
		{
			Name:   "empty",
			String: "constraints not satisfiable",
			Error:  NotSatisfiable{},
		},
		{
			Name: "single failure",
			Error: NotSatisfiable{
				Mandatory("a"),
			},
			String: fmt.Sprintf("constraints not satisfiable: %s",
				Mandatory("a").String()),
		},
		{
			Name: "multiple failures",
			Error: NotSatisfiable{
				Mandatory("a"),
				Prohibited("b"),
			},
			String: fmt.Sprintf("constraints not satisfiable: %s, %s",
				Mandatory("a").String(), Prohibited("b").String()),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.String, tt.Error.Error())
		})
	}
}

func TestSolve(t *testing.T) {
	type tc struct {
		Name        string
		Constraints []Constraint
		Installed   []Identifier
		Error       error
	}

	for _, tt := range []tc{
		{
			Name:      "no variables",
			Installed: []Identifier{},
		},
		{
			Name:        "single mandatory variable is installed",
			Constraints: []Constraint{Mandatory("a")},
			Installed:   []Identifier{"a"},
		},
		{
			Name:        "both mandatory and prohibited produce error",
			Constraints: []Constraint{Mandatory("a"), Prohibited("a")},
			Error: NotSatisfiable{
				Mandatory("a"),
				Prohibited("a"),
			},
		},
		{
			Name: "dependency is installed",
			Constraints: []Constraint{
				Mandatory("b"),
				Dependency("b", "a"),
			},
			Installed: []Identifier{"a", "b"},
		},
		{
			Name: "transitive dependency is installed",
			Constraints: []Constraint{
				Dependency("b", "a"),
				Mandatory("c"),
				Dependency("c", "b"),
			},
			Installed: []Identifier{"a", "b", "c"},
		},
		{
			Name: "both dependencies are installed",
			Constraints: []Constraint{
				Mandatory("c"),
				Dependency("c", "a"),
				Dependency("c", "b"),
			},
			Installed: []Identifier{"a", "b", "c"},
		},
		{
			Name: "solution with first dependency is selected",
			Constraints: []Constraint{
				Conflict("b", "a"),
				Mandatory("c"),
				Dependency("c", "a", "b"),
			},
			Installed: []Identifier{"a", "c"},
		},
		{
			Name: "solution with only first dependency is selected",
			Constraints: []Constraint{
				Mandatory("c"),
				Dependency("c", "a", "b"),
			},
			Installed: []Identifier{"a", "c"},
		},
		{
			Name: "solution with first dependency is selected (reverse)",
			Constraints: []Constraint{
				Conflict("b", "a"),
				Mandatory("c"),
				Dependency("c", "b", "a"),
			},
			Installed: []Identifier{"b", "c"},
		},
		{
			Name: "two mandatory but conflicting packages",
			Constraints: []Constraint{
				Mandatory("b"),
				Mandatory("a"),
				Conflict("b", "a"),
			},
			Error: NotSatisfiable{
				Mandatory("a"),
				Mandatory("b"),
				Conflict("b", "a"),
			},
		},
		{
			Name: "irrelevant dependencies don't influence search order",
			Constraints: []Constraint{
				Dependency("a", "x", "y"),
				Mandatory("b"),
				Dependency("b", "y", "x"),
			},
			Installed: []Identifier{"b", "y"},
		},
		{
			Name: "cardinality constraint prevents resolution",
			Constraints: []Constraint{
				Mandatory("a"),
				Dependency("a", "x", "y"),
				Mandatory("x"),
				Mandatory("y"),
				AtMost("a", 1, "x", "y"),
			},
			Error: NotSatisfiable{
				Mandatory("y"),
				Mandatory("x"),
				AtMost("a", 1, "x", "y"),
			},
		},
		{
			Name: "cardinality constraint forces alternative",
			Constraints: []Constraint{
				Mandatory("a"),
				Dependency("a", "x", "y"),
				AtMost("a", 1, "x", "y"),
				Mandatory("b"),
				Dependency("b", "y"),
			},
			Installed: []Identifier{"a", "b", "y"},
		},
		{
			Name: "two dependencies satisfied by one variable",
			Constraints: []Constraint{
				Mandatory("a"),
				Dependency("a", "y"),
				Mandatory("b"),
				Dependency("b", "x", "y"),
			},
			Installed: []Identifier{"a", "b", "y"},
		},
		{
			Name: "foo two dependencies satisfied by one variable",
			Constraints: []Constraint{
				Mandatory("a"),
				Dependency("a", "y", "z", "m"),
				Mandatory("b"),
				Dependency("b", "x", "y"),
			},
			Installed: []Identifier{"a", "b", "y"},
		},
		{
			Name: "result size larger than minimum due to preference",
			Constraints: []Constraint{
				Mandatory("a"),
				Dependency("a", "x", "y"),
				Mandatory("b"),
				Dependency("b", "y"),
			},
			Installed: []Identifier{"a", "b", "x", "y"},
		},
		{
			Name: "only the least preferable choice is acceptable",
			Constraints: []Constraint{
				Mandatory("a"),
				Dependency("a", "a1", "a2"),
				Conflict("a1", "c1"),
				Conflict("a1", "c2"),
				Conflict("a2", "c1"),
				Mandatory("b"),
				Dependency("b", "b1", "b2"),
				Conflict("b1", "c1"),
				Conflict("b1", "c2"),
				Conflict("b2", "c1"),
				Mandatory("c"),
				Dependency("c", "c1", "c2"),
			},
			Installed: []Identifier{"a", "a2", "b", "b2", "c", "c2"},
		},
		{
			Name: "preferences respected with multiple dependencies per variable",
			Constraints: []Constraint{
				Mandatory("a"),
				Dependency("a", "x1", "x2"),
				Dependency("a", "y1", "y2"),
			},
			Installed: []Identifier{"a", "x1", "y1"},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)

			var traces bytes.Buffer
			s, err := New(WithInput(tt.Constraints), WithTracer(LoggingTracer{Writer: &traces}))
			if err != nil {
				t.Fatalf("failed to initialize solver: %s", err)
			}

			installed, err := s.Solve(context.TODO())

			if installed != nil {
				sort.SliceStable(installed, func(i, j int) bool {
					return installed[i] < installed[j]
				})
			}

			assert.Equal(tt.Installed, installed)
			assert.Equal(tt.Error, err)

			if t.Failed() {
				t.Logf("\n%s", traces.String())
			}
		})
	}
}
