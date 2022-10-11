package solver

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
)

var BenchmarkConstraints = func() []Constraint {
	const (
		length      = 256
		seed        = 9
		pMandatory  = .1
		pDependency = .15
		nDependency = 6
		pConflict   = .05
		nConflict   = 3
	)

	id := func(i int) Identifier {
		return Identifier(strconv.Itoa(i))
	}

	input := func(i int) []Constraint {
		var c []Constraint
		subject := id(i)
		if rand.Float64() < pMandatory {
			c = append(c, Mandatory(subject))
		}
		if rand.Float64() < pDependency {
			n := rand.Intn(nDependency-1) + 1
			var d []Identifier
			for x := 0; x < n; x++ {
				y := i
				for y == i {
					y = rand.Intn(length)
				}
				d = append(d, id(y))
			}
			c = append(c, Dependency(subject, d...))
		}
		if rand.Float64() < pConflict {
			n := rand.Intn(nConflict-1) + 1
			for x := 0; x < n; x++ {
				y := i
				for y == i {
					y = rand.Intn(length)
				}
				c = append(c, Conflict(subject, id(y)))
			}
		}
		return c
	}

	rand.Seed(seed)
	constraints := make([]Constraint, 0)
	for i := 0; i < length; i++ {
		constrs := input(i)
		constraints = append(constraints, constrs...)
	}
	return constraints
}()

func BenchmarkSolve(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s, err := New(WithInput(BenchmarkConstraints))
		if err != nil {
			b.Fatalf("failed to initialize solver: %s", err)
		}
		s.Solve(context.Background())
	}
}

func BenchmarkNewInput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := New(WithInput(BenchmarkConstraints))
		if err != nil {
			b.Fatalf("failed to initialize solver: %s", err)
		}
	}
}
