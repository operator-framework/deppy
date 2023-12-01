package solver

import (
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

	random := rand.New(rand.NewSource(seed)) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.

	id := func(i int) deppy.Identifier {
		return deppy.Identifier(strconv.Itoa(i))
	}

	variable := func(i int) TestVariable {
		var c []deppy.Constraint
		if random.Float64() < pMandatory {
			c = append(c, constraint.Mandatory())
		}
		if random.Float64() < pDependency {
			n := random.Intn(nDependency-1) + 1
			var d []deppy.Identifier
			for x := 0; x < n; x++ {
				y := i
				for y == i {
					y = random.Intn(length)
				}
				d = append(d, id(y))
			}
			c = append(c, constraint.Dependency(d...))
		}
		if random.Float64() < pConflict {
			n := random.Intn(nConflict-1) + 1
			for x := 0; x < n; x++ {
				y := i
				for y == i {
					y = random.Intn(length)
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
		s, err := New()
		if err != nil {
			b.Fatalf("failed to initialize solver: %s", err)
		}
		_, err = s.Solve(BenchmarkInput)
		if err != nil {
			b.Fatalf("failed to initialize solver: %s", err)
		}
	}
}
