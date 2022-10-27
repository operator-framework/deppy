package sat_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/internal/sat"
	pkgsat "github.com/operator-framework/deppy/pkg/sat"
	"github.com/operator-framework/deppy/pkg/sat/constraints"
)

type TestVariable struct {
	identifier  pkgsat.Identifier
	constraints []pkgsat.Constraint
}

func (i TestVariable) Identifier() pkgsat.Identifier {
	return i.identifier
}

func (i TestVariable) Constraints() []pkgsat.Constraint {
	return i.constraints
}

func (i TestVariable) GoString() string {
	return fmt.Sprintf("%q", i.Identifier())
}

func variable(id pkgsat.Identifier, constraints ...pkgsat.Constraint) pkgsat.Variable {
	return TestVariable{
		identifier:  id,
		constraints: constraints,
	}
}

func TestNotSatisfiableError(t *testing.T) {
	type tc struct {
		Name   string
		Error  sat.NotSatisfiable
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
			Error:  sat.NotSatisfiable{},
		},
		{
			Name: "single failure",
			Error: sat.NotSatisfiable{
				pkgsat.AppliedConstraint{
					Variable:   variable("a", constraints.Mandatory()),
					Constraint: constraints.Mandatory(),
				},
			},
			String: fmt.Sprintf("constraints not satisfiable: %s",
				constraints.Mandatory().String("a")),
		},
		{
			Name: "multiple failures",
			Error: sat.NotSatisfiable{
				pkgsat.AppliedConstraint{
					Variable:   variable("a", constraints.Mandatory()),
					Constraint: constraints.Mandatory(),
				},
				pkgsat.AppliedConstraint{
					Variable:   variable("b", constraints.Prohibited()),
					Constraint: constraints.Prohibited(),
				},
			},
			String: fmt.Sprintf("constraints not satisfiable: %s, %s",
				constraints.Mandatory().String("a"), constraints.Prohibited().String("b")),
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
		Variables []pkgsat.Variable
		Installed []pkgsat.Identifier
		Error     error
	}

	for _, tt := range []tc{
		{
			Name: "no variables",
		},
		{
			Name:      "unnecessary variable is not installed",
			Variables: []pkgsat.Variable{variable("a")},
		},
		{
			Name:      "single mandatory variable is installed",
			Variables: []pkgsat.Variable{variable("a", constraints.Mandatory())},
			Installed: []pkgsat.Identifier{"a"},
		},
		{
			Name:      "both mandatory and prohibited produce error",
			Variables: []pkgsat.Variable{variable("a", constraints.Mandatory(), constraints.Prohibited())},
			Error: sat.NotSatisfiable{
				{
					Variable:   variable("a", constraints.Mandatory(), constraints.Prohibited()),
					Constraint: constraints.Mandatory(),
				},
				{
					Variable:   variable("a", constraints.Mandatory(), constraints.Prohibited()),
					Constraint: constraints.Prohibited(),
				},
			},
		},
		{
			Name: "dependency is installed",
			Variables: []pkgsat.Variable{
				variable("a"),
				variable("b", constraints.Mandatory(), constraints.Dependency("a")),
			},
			Installed: []pkgsat.Identifier{"a", "b"},
		},
		{
			Name: "transitive dependency is installed",
			Variables: []pkgsat.Variable{
				variable("a"),
				variable("b", constraints.Dependency("a")),
				variable("c", constraints.Mandatory(), constraints.Dependency("b")),
			},
			Installed: []pkgsat.Identifier{"a", "b", "c"},
		},
		{
			Name: "both dependencies are installed",
			Variables: []pkgsat.Variable{
				variable("a"),
				variable("b"),
				variable("c", constraints.Mandatory(), constraints.Dependency("a"), constraints.Dependency("b")),
			},
			Installed: []pkgsat.Identifier{"a", "b", "c"},
		},
		{
			Name: "solution with first dependency is selected",
			Variables: []pkgsat.Variable{
				variable("a"),
				variable("b", constraints.Conflict("a")),
				variable("c", constraints.Mandatory(), constraints.Dependency("a", "b")),
			},
			Installed: []pkgsat.Identifier{"a", "c"},
		},
		{
			Name: "solution with only first dependency is selected",
			Variables: []pkgsat.Variable{
				variable("a"),
				variable("b"),
				variable("c", constraints.Mandatory(), constraints.Dependency("a", "b")),
			},
			Installed: []pkgsat.Identifier{"a", "c"},
		},
		{
			Name: "solution with first dependency is selected (reverse)",
			Variables: []pkgsat.Variable{
				variable("a"),
				variable("b", constraints.Conflict("a")),
				variable("c", constraints.Mandatory(), constraints.Dependency("b", "a")),
			},
			Installed: []pkgsat.Identifier{"b", "c"},
		},
		{
			Name: "two mandatory but conflicting packages",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory()),
				variable("b", constraints.Mandatory(), constraints.Conflict("a")),
			},
			Error: sat.NotSatisfiable{
				{
					Variable:   variable("a", constraints.Mandatory()),
					Constraint: constraints.Mandatory(),
				},
				{
					Variable:   variable("b", constraints.Mandatory(), constraints.Conflict("a")),
					Constraint: constraints.Mandatory(),
				},
				{
					Variable:   variable("b", constraints.Mandatory(), constraints.Conflict("a")),
					Constraint: constraints.Conflict("a"),
				},
			},
		},
		{
			Name: "irrelevant dependencies don't influence search order",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Dependency("x", "y")),
				variable("b", constraints.Mandatory(), constraints.Dependency("y", "x")),
				variable("x"),
				variable("y"),
			},
			Installed: []pkgsat.Identifier{"b", "y"},
		},
		{
			Name: "cardinality constraint prevents resolution",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("x", "y"), constraints.AtMost(1, "x", "y")),
				variable("x", constraints.Mandatory()),
				variable("y", constraints.Mandatory()),
			},
			Error: sat.NotSatisfiable{
				{
					Variable:   variable("a", constraints.Mandatory(), constraints.Dependency("x", "y"), constraints.AtMost(1, "x", "y")),
					Constraint: constraints.AtMost(1, "x", "y"),
				},
				{
					Variable:   variable("x", constraints.Mandatory()),
					Constraint: constraints.Mandatory(),
				},
				{
					Variable:   variable("y", constraints.Mandatory()),
					Constraint: constraints.Mandatory(),
				},
			},
		},
		{
			Name: "cardinality constraint forces alternative",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("x", "y"), constraints.AtMost(1, "x", "y")),
				variable("b", constraints.Mandatory(), constraints.Dependency("y")),
				variable("x"),
				variable("y"),
			},
			Installed: []pkgsat.Identifier{"a", "b", "y"},
		},
		{
			Name: "two dependencies satisfied by one variable",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("y")),
				variable("b", constraints.Mandatory(), constraints.Dependency("x", "y")),
				variable("x"),
				variable("y"),
			},
			Installed: []pkgsat.Identifier{"a", "b", "y"},
		},
		{
			Name: "foo two dependencies satisfied by one variable",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("y", "z", "m")),
				variable("b", constraints.Mandatory(), constraints.Dependency("x", "y")),
				variable("x"),
				variable("y"),
				variable("z"),
				variable("m"),
			},
			Installed: []pkgsat.Identifier{"a", "b", "y"},
		},
		{
			Name: "result size larger than minimum due to preference",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("x", "y")),
				variable("b", constraints.Mandatory(), constraints.Dependency("y")),
				variable("x"),
				variable("y"),
			},
			Installed: []pkgsat.Identifier{"a", "b", "x", "y"},
		},
		{
			Name: "only the least preferable choice is acceptable",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("a1", "a2")),
				variable("a1", constraints.Conflict("c1"), constraints.Conflict("c2")),
				variable("a2", constraints.Conflict("c1")),
				variable("b", constraints.Mandatory(), constraints.Dependency("b1", "b2")),
				variable("b1", constraints.Conflict("c1"), constraints.Conflict("c2")),
				variable("b2", constraints.Conflict("c1")),
				variable("c", constraints.Mandatory(), constraints.Dependency("c1", "c2")),
				variable("c1"),
				variable("c2"),
			},
			Installed: []pkgsat.Identifier{"a", "a2", "b", "b2", "c", "c2"},
		},
		{
			Name: "preferences respected with multiple dependencies per variable",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("x1", "x2"), constraints.Dependency("y1", "y2")),
				variable("x1"),
				variable("x2"),
				variable("y1"),
				variable("y2"),
			},
			Installed: []pkgsat.Identifier{"a", "x1", "y1"},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)

			var traces bytes.Buffer
			s, err := sat.NewSolver(sat.WithInput(tt.Variables), sat.WithTracer(sat.LoggingTracer{Writer: &traces}))
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
			// increasing order of the identifier of the
			// constraint's variable, with ties broken
			// in favor of the constraint that appears
			// earliest in the variable's list of
			// constraints.
			var ns sat.NotSatisfiable
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

			var ids []pkgsat.Identifier
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
	_, err := sat.NewSolver(sat.WithInput([]pkgsat.Variable{
		variable("a"),
		variable("a"),
	}))
	assert.Equal(t, pkgsat.DuplicateIdentifier("a"), err)
}
