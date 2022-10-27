package sat_test

import (
	"context"
	"math/rand"
	"strconv"
	"testing"

	"github.com/operator-framework/deppy/internal/sat"
	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
)

var BenchmarkInput = func() []pkgconstraints.IVariable {
	const (
		length      = 256
		seed        = 9
		pMandatory  = .1
		pDependency = .15
		nDependency = 6
		pConflict   = .05
		nConflict   = 3
	)

	id := func(i int) pkgconstraints.Identifier {
		return pkgconstraints.Identifier(strconv.Itoa(i))
	}

	variable := func(i int) TestVariable {
		var c []pkgconstraints.Constraint
		if rand.Float64() < pMandatory {
			c = append(c, pkgconstraints.Mandatory())
		}
		if rand.Float64() < pDependency {
			n := rand.Intn(nDependency-1) + 1
			var d []pkgconstraints.Identifier
			for x := 0; x < n; x++ {
				y := i
				for y == i {
					y = rand.Intn(length)
				}
				d = append(d, id(y))
			}
			c = append(c, pkgconstraints.Dependency(d...))
		}
		if rand.Float64() < pConflict {
			n := rand.Intn(nConflict-1) + 1
			for x := 0; x < n; x++ {
				y := i
				for y == i {
					y = rand.Intn(length)
				}
				c = append(c, pkgconstraints.Conflict(id(y)))
			}
		}
		return TestVariable{
			identifier:  id(i),
			constraints: c,
		}
	}

	rand.Seed(seed)
	result := make([]pkgconstraints.IVariable, length)
	for i := range result {
		result[i] = variable(i)
	}
	return result
}()

func BenchmarkSolve(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s, err := sat.NewSolver(sat.WithInput(BenchmarkInput))
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
		_, err := sat.NewSolver(sat.WithInput(BenchmarkInput))
		if err != nil {
			b.Fatalf("failed to initialize solver: %s", err)
		}
	}
}
