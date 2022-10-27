//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o zz_search_test.go ../../../../../vendor/github.com/go-air/gini/inter S

package sat_test

import (
	"context"
	"testing"

	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/z"
	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/internal/sat"
	pkgsat "github.com/operator-framework/deppy/pkg/sat"
	"github.com/operator-framework/deppy/pkg/sat/constraints"
)

type TestScopeCounter struct {
	depth *int
	inter.S
}

func (c *TestScopeCounter) Test(dst []z.Lit) (result int, out []z.Lit) {
	result, out = c.S.Test(dst)
	*c.depth++
	return
}

func (c *TestScopeCounter) Untest() (result int) {
	result = c.S.Untest()
	*c.depth--
	return
}

func TestSearch(t *testing.T) {
	type tc struct {
		Name          string
		Variables     []pkgsat.Variable
		TestReturns   []int
		UntestReturns []int
		Result        int
		Assumptions   []pkgsat.Identifier
	}

	for _, tt := range []tc{
		{
			Name: "children popped from back of deque when guess popped",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("c")),
				variable("b", constraints.Mandatory()),
				variable("c"),
			},
			TestReturns:   []int{0, -1},
			UntestReturns: []int{-1, -1},
			Result:        -1,
			Assumptions:   nil,
		},
		{
			Name: "candidates exhausted",
			Variables: []pkgsat.Variable{
				variable("a", constraints.Mandatory(), constraints.Dependency("x")),
				variable("b", constraints.Mandatory(), constraints.Dependency("y")),
				variable("x"),
				variable("y"),
			},
			TestReturns:   []int{0, 0, -1, 1},
			UntestReturns: []int{0},
			Result:        1,
			Assumptions:   []pkgsat.Identifier{"a", "b", "y"},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)

			var s sat.FakeS
			for i, result := range tt.TestReturns {
				s.TestReturnsOnCall(i, result, nil)
			}
			for i, result := range tt.UntestReturns {
				s.UntestReturnsOnCall(i, result)
			}

			var depth int
			counter := &TestScopeCounter{depth: &depth, S: &s}

			lits, err := pkgsat.NewLitMapping(tt.Variables)
			assert.NoError(err)
			h := sat.Search{
				S:      counter,
				Slits:  lits,
				Tracer: sat.DefaultTracer{},
			}

			var anchors []z.Lit
			for _, id := range h.Slits.AnchorIdentifiers() {
				anchors = append(anchors, h.Slits.LitOf(id))
			}

			result, ms, _ := h.Do(context.Background(), anchors)

			assert.Equal(tt.Result, result)
			var ids []pkgsat.Identifier
			for _, m := range ms {
				ids = append(ids, lits.VariableOf(m).Identifier())
			}
			assert.Equal(tt.Assumptions, ids)
			assert.Equal(0, depth)
		})
	}
}
