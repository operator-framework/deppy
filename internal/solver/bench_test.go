package solver

import (
	"context"
	"math/rand"
	"strconv"
	"testing"

	"github.com/operator-framework/deppy/pkg/deppy"
	"github.com/operator-framework/deppy/pkg/deppy/constraint"
)

var BenchmarkInput = func() []deppy.Variable {
	const (
		length      = 256
		seed        = 9
		pMandatory  = .1
		pDependency = .15
		nDependency = 6
		pConflict   = .05
		nConflict   = 3
	)

	id := func(i int) deppy.Identifier {
		return deppy.Identifier(strconv.Itoa(i))
	}

	variable := func(i int) TestVariable {
		var c []deppy.Constraint
		if rand.Float64() < pMandatory {
			c = append(c, constraint.Mandatory())
		}
		if rand.Float64() < pDependency {
			n := rand.Intn(nDependency-1) + 1
			var d []deppy.Identifier
			for x := 0; x < n; x++ {
				y := i
				for y == i {
					y = rand.Intn(length)
				}
				d = append(d, id(y))
			}
			c = append(c, constraint.Dependency(d...))
		}
		if rand.Float64() < pConflict {
			n := rand.Intn(nConflict-1) + 1
			for x := 0; x < n; x++ {
				y := i
				for y == i {
					y = rand.Intn(length)
				}
				c = append(c, constraint.Conflict(id(y)))
			}
		}
		return TestVariable{
			identifier:  id(i),
			constraints: c,
		}
	}

	result := make([]deppy.Variable, length)
	for i := range result {
		result[i] = variable(i)
	}
	return result
}()

func BenchmarkSolve(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s, err := NewSolver(WithInput(BenchmarkInput))
		if err != nil {
			b.Fatalf("failed to initialize solver: %s", err)
		}
		_, err = s.Solve(context.Background())
		if err != nil {
			b.Fatalf("failed to initialize solver: %s", err)
		}
	}
}

func BenchmarkNewInput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewSolver(WithInput(BenchmarkInput))
		if err != nil {
			b.Fatalf("failed to initialize solver: %s", err)
		}
	}
}
