//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o zz_search_test.go ../../../../../vendor/github.com/go-air/gini/inter S

package solver

import (
	"context"
	"testing"

	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/z"
	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/pkg/deppy/constraint"

	"github.com/operator-framework/deppy/pkg/deppy"
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
		Variables     []deppy.Variable
		TestReturns   []int
		UntestReturns []int
		Result        int
		Assumptions   []deppy.Identifier
	}

	for _, tt := range []tc{
		{
			Name: "children popped from back of deque when guess popped",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory(), constraint.Dependency("c")),
				variable("b", constraint.Mandatory()),
				variable("c"),
			},
			TestReturns:   []int{0, -1},
			UntestReturns: []int{-1, -1},
			Result:        -1,
			Assumptions:   nil,
		},
		{
			Name: "candidates exhausted",
			Variables: []deppy.Variable{
				variable("a", constraint.Mandatory(), constraint.Dependency("x")),
				variable("b", constraint.Mandatory(), constraint.Dependency("y")),
				variable("x"),
				variable("y"),
			},
			TestReturns:   []int{0, 0, -1, 1},
			UntestReturns: []int{0},
			Result:        1,
			Assumptions:   []deppy.Identifier{"a", "b", "y"},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)

			var s FakeS
			for i, result := range tt.TestReturns {
				s.TestReturnsOnCall(i, result, nil)
			}
			for i, result := range tt.UntestReturns {
				s.UntestReturnsOnCall(i, result)
			}

			var depth int
			counter := &TestScopeCounter{depth: &depth, S: &s}

			lits, err := newLitMapping(tt.Variables)
			assert.NoError(err)
			h := search{
				s:      counter,
				lits:   lits,
				tracer: DefaultTracer{},
			}

			var anchors []z.Lit
			for _, id := range h.lits.AnchorIdentifiers() {
				anchors = append(anchors, h.lits.LitOf(id))
			}

			result, ms, _ := h.Do(context.Background(), anchors)

			assert.Equal(tt.Result, result)
			var ids []deppy.Identifier
			for _, m := range ms {
				ids = append(ids, lits.VariableOf(m).Identifier())
			}
			assert.Equal(tt.Assumptions, ids)
			assert.Equal(0, depth)
		})
	}
}
