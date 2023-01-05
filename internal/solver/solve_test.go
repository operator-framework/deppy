package solver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/pkg/deppy/constraint"

	"github.com/operator-framework/deppy/pkg/deppy"
)

type TestVariable struct {
	identifier  deppy.Identifier
	constraints []deppy.Constraint
}

func (i TestVariable) Identifier() deppy.Identifier {
	return i.identifier
}

func (i TestVariable) Constraints() []deppy.Constraint {
	return i.constraints
}

func (i TestVariable) GoString() string {
	return fmt.Sprintf("%q", i.Identifier())
}

func variable(id deppy.Identifier, constraints ...deppy.Constraint) deppy.Variable {
	return TestVariable{
		identifier:  id,
		constraints: constraints,
	}
}

func TestNotSatisfiableError(t *testing.T) {
	type tc struct {
		Name   string
		Error  deppy.NotSatisfiable
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
			Error:  deppy.NotSatisfiable{},
		},
		{
			Name: "single failure",
			Error: deppy.NotSatisfiable{
				deppy.AppliedConstraint{
					Variable:   variable("a", constraint.Mandatory()),
					Constraint: constraint.Mandatory(),
				},
			},
			String: fmt.Sprintf("constraints not satisfiable: %s",
				constraint.Mandatory().String("a")),
		},
		{
			Name: "multiple failures",
			Error: deppy.NotSatisfiable{
				deppy.AppliedConstraint{
					Variable:   variable("a", constraint.Mandatory()),
					Constraint: constraint.Mandatory(),
				},
				deppy.AppliedConstraint{
					Variable:   variable("b", constraint.Prohibited()),
					Constraint: constraint.Prohibited(),
				},
			},
			String: fmt.Sprintf("constraints not satisfiable: %s, %s",
				constraint.Mandatory().String("a"), constraint.Prohibited().String("b")),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.String, tt.Error.Error())
		})
	}
}

func TestSolve(t *testing.T) {
	type tc struct {
		Name      string
		Variables []deppy.Variable
		Installed []deppy.Identifier
		Error     error
	}

	for _, tt := range []tc{
		{
			Name: "no variables",
		},
		{
			Name:      "unnecessary variable is not installed",
			Variables: []deppy.Variable{variable("a")},
		},
		{
			Name:      "single mandatory variable is installed",
			Variables: []deppy.Variable{variable("a", constraint.Mandatory())},
			Installed: []deppy.Identifier{"a"},
		},
		{
			Name:      "both mandatory and prohibited produce error",
			Variables: []deppy.Variable{variable("a", constraint.Mandatory(), constraint.Prohibited())},
			Error: deppy.NotSatisfiable{
				{
					Variable:   variable("a", constraint.Mandatory(), constraint.Prohibited()),
					Constraint: constraint.Mandatory(),
				},
				{
					Variable:   variable("a", constraint.Mandatory(), constraint.Prohibited()),
					Constraint: constraint.Prohibited(),
				},
			},
		},
		{
			Name: "dependency is installed",
			Variables: []deppy.Variable{
				variable("a"),
				variable("b", constraint.Mandatory(), constraint.Dependency("a")),
			},
			Installed: []deppy.Identifier{"a", "b"},
		},
		{
			Name: "transitive dependency is installed",
			Variables: []deppy.Variable{
				variable("a"),
				variable("b", constraint.Dependency("a")),
				variable("c", constraint.Mandatory(), constraint.Dependency("b")),
			},
			Installed: []deppy.Identifier{"a", "b", "c"},
		},
		{
			Name: "both dependencies are installed",
			Variables: []deppy.Variable{
				variable("a"),
				variable("b"),
				variable("c", constraint.Mandatory(), constraint.Dependency("a"), constraint.Dependency("b")),
			},
			Installed: []deppy.Identifier{"a", "b", "c"},
		},
		{
			Name: "solution with first dependency is selected",
			Variables: []deppy.Variable{
				variable("a"),
				variable("b", constraint.Conflict("a")),
				variable("c", constraint.Mandatory(), constraint.Dependency("a", "b")),
			},
			Installed: []deppy.Identifier{"a", "c"},
		},
		{
			Name: "solution with only first dependency is selected",
			Variables: []deppy.Variable{
				variable("a"),
				variable("b"),
				variable("c", constraint.Mandatory(), constraint.Dependency("a", "b")),
			},
			Installed: []deppy.Identifier{"a", "c"},
		},
		{
			Name: "solution with first dependency is selected (reverse)",
			Variables: []deppy.Variable{
				variable("a"),
				variable("b", constraint.Conflict("a")),
				variable("c", constraint.Mandatory(), constraint.Dependency("b", "a")),
			},
			Installed: []deppy.Identifier{"b", "c"},
		},
		{
			Name: "two mandatory but conflicting packages",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory()),
				variable("b", constraint.Mandatory(), constraint.Conflict("a")),
			},
			Error: deppy.NotSatisfiable{
				{
					Variable:   variable("a", constraint.Mandatory()),
					Constraint: constraint.Mandatory(),
				},
				{
					Variable:   variable("b", constraint.Mandatory(), constraint.Conflict("a")),
					Constraint: constraint.Mandatory(),
				},
				{
					Variable:   variable("b", constraint.Mandatory(), constraint.Conflict("a")),
					Constraint: constraint.Conflict("a"),
				},
			},
		},
		{
			Name: "irrelevant dependencies don't influence search Order",
			Variables: []deppy.Variable{
				variable("a", constraint.Dependency("x", "y")),
				variable("b", constraint.Mandatory(), constraint.Dependency("y", "x")),
				variable("x"),
				variable("y"),
			},
			Installed: []deppy.Identifier{"b", "y"},
		},
		{
			Name: "cardinality constraint prevents resolution",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory(), constraint.Dependency("x", "y"), constraint.AtMost(1, "x", "y")),
				variable("x", constraint.Mandatory()),
				variable("y", constraint.Mandatory()),
			},
			Error: deppy.NotSatisfiable{
				{
					Variable:   variable("a", constraint.Mandatory(), constraint.Dependency("x", "y"), constraint.AtMost(1, "x", "y")),
					Constraint: constraint.AtMost(1, "x", "y"),
				},
				{
					Variable:   variable("x", constraint.Mandatory()),
					Constraint: constraint.Mandatory(),
				},
				{
					Variable:   variable("y", constraint.Mandatory()),
					Constraint: constraint.Mandatory(),
				},
			},
		},
		{
			Name: "cardinality constraint forces alternative",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory(), constraint.Dependency("x", "y"), constraint.AtMost(1, "x", "y")),
				variable("b", constraint.Mandatory(), constraint.Dependency("y")),
				variable("x"),
				variable("y"),
			},
			Installed: []deppy.Identifier{"a", "b", "y"},
		},
		{
			Name: "foo two dependencies satisfied by one variable",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory(), constraint.Dependency("y", "z", "m")),
				variable("b", constraint.Mandatory(), constraint.Dependency("x", "y")),
				variable("x"),
				variable("y"),
				variable("z"),
				variable("m"),
			},
			Installed: []deppy.Identifier{"a", "b", "y"},
		},
		{
			Name: "result size larger than minimum due to preference",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory(), constraint.Dependency("x", "y")),
				variable("b", constraint.Mandatory(), constraint.Dependency("y")),
				variable("x"),
				variable("y"),
			},
			Installed: []deppy.Identifier{"a", "b", "x", "y"},
		},
		{
			Name: "only the least preferable choice is acceptable",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory(), constraint.Dependency("a1", "a2")),
				variable("a1", constraint.Conflict("c1"), constraint.Conflict("c2")),
				variable("a2", constraint.Conflict("c1")),
				variable("b", constraint.Mandatory(), constraint.Dependency("b1", "b2")),
				variable("b1", constraint.Conflict("c1"), constraint.Conflict("c2")),
				variable("b2", constraint.Conflict("c1")),
				variable("c", constraint.Mandatory(), constraint.Dependency("c1", "c2")),
				variable("c1"),
				variable("c2"),
			},
			Installed: []deppy.Identifier{"a", "a2", "b", "b2", "c", "c2"},
		},
		{
			Name: "preferences respected with multiple dependencies per variable",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory(), constraint.Dependency("x1", "x2"), constraint.Dependency("y1", "y2")),
				variable("x1"),
				variable("x2"),
				variable("y1"),
				variable("y2"),
			},
			Installed: []deppy.Identifier{"a", "x1", "y1"},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)

			var traces bytes.Buffer
			s, err := NewSolver(WithInput(tt.Variables), WithTracer(LoggingTracer{Writer: &traces}))
			if err != nil {
				t.Fatalf("failed to initialize solver: %s", err)
			}

			installed, err := s.Solve(context.TODO())

			if installed != nil {
				sort.SliceStable(installed, func(i, j int) bool {
					return installed[i].Identifier() < installed[j].Identifier()
				})
			}

			// Failed constraints are sorted in lexically
			// increasing Order of the identifier of the
			// constraint's variable, with ties broken
			// in favor of the constraint that appears
			// earliest in the variable's list of
			// constraints.
			var ns deppy.NotSatisfiable
			if errors.As(err, &ns) {
				sort.SliceStable(ns, func(i, j int) bool {
					if ns[i].Variable.Identifier() != ns[j].Variable.Identifier() {
						return ns[i].Variable.Identifier() < ns[j].Variable.Identifier()
					}
					var x, y int
					for ii, c := range ns[i].Variable.Constraints() {
						if reflect.DeepEqual(c, ns[i].Constraint) {
							x = ii
							break
						}
					}
					for ij, c := range ns[j].Variable.Constraints() {
						if reflect.DeepEqual(c, ns[j].Constraint) {
							y = ij
							break
						}
					}
					return x < y
				})
			}

			var ids []deppy.Identifier
			for _, variable := range installed {
				ids = append(ids, variable.Identifier())
			}
			assert.Equal(tt.Installed, ids)
			assert.Equal(tt.Error, err)

			if t.Failed() {
				t.Logf("\n%s", traces.String())
			}
		})
	}
}

func TestDuplicateIdentifier(t *testing.T) {
	_, err := NewSolver(WithInput([]deppy.Variable{
		variable("a"),
		variable("a"),
	}))
	assert.Equal(t, DuplicateIdentifier("a"), err)
}
