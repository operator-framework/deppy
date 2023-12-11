package solver

import (
	"errors"
	"fmt"

	"github.com/go-air/gini"
	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/z"

	"github.com/operator-framework/deppy/pkg/deppy"
)

type Solver struct {
	tracer deppy.Tracer
}

const (
	satisfiable   = 1
	unsatisfiable = -1
	unknown       = 0
)

// Solve takes a slice containing all Variables and returns a slice
// containing only those Variables that were selected for
// installation. If no solution is possible an error is returned.
func (s *Solver) Solve(input []deppy.Variable) ([]deppy.Variable, error) {
	giniSolver := gini.New()
	litMap, err := newLitMapping(input)
	if err != nil {
		return nil, err
	}

	result, err := s.solve(giniSolver, litMap)

	// This likely indicates a bug, so discard whatever
	// return values were produced.
	if derr := litMap.Error(); derr != nil {
		return nil, derr
	}

	return result, err
}

func (s *Solver) solve(giniSolver inter.S, litMap *litMapping) ([]deppy.Variable, error) {
	// teach all constraints to the solver
	litMap.AddConstraints(giniSolver)

	// collect literals of all mandatory variables to assume as a baseline
	anchors := litMap.AnchorIdentifiers()
	assumptions := make([]z.Lit, len(anchors))
	for i := range anchors {
		assumptions[i] = litMap.LitOf(anchors[i])
	}

	// assume that all constraints hold
	litMap.AssumeConstraints(giniSolver)
	giniSolver.Assume(assumptions...)

	var buffer []z.Lit
	var aset map[z.Lit]struct{}
	// push a new test scope with the baseline assumptions, to prevent them from being cleared during search
	outcome, _ := giniSolver.Test(nil)
	if outcome != satisfiable && outcome != unsatisfiable {
		// searcher for solutions in input Order, so that preferences
		// can be taken into account (i.e. prefer one catalog to another)
		outcome, assumptions, aset = (&search{s: giniSolver, lits: litMap, tracer: s.tracer}).Do(assumptions)
	}
	switch outcome {
	case satisfiable:
		buffer = litMap.Lits(buffer)
		var extras, excluded []z.Lit
		for _, m := range buffer {
			if _, ok := aset[m]; ok {
				continue
			}
			if !giniSolver.Value(m) {
				excluded = append(excluded, m.Not())
				continue
			}
			extras = append(extras, m)
		}
		giniSolver.Untest()
		cs := litMap.CardinalityConstrainer(giniSolver, extras)
		giniSolver.Assume(assumptions...)
		giniSolver.Assume(excluded...)
		litMap.AssumeConstraints(giniSolver)
		giniSolver.Test(nil)
		for w := 0; w <= cs.N(); w++ {
			giniSolver.Assume(cs.Leq(w))
			if giniSolver.Solve() == satisfiable {
				return litMap.Variables(giniSolver), nil
			}
		}
		// Something is wrong if we can't find a model anymore
		// after optimizing for cardinality.
		return nil, fmt.Errorf("unexpected internal error")
	case unsatisfiable:
		return nil, deppy.NotSatisfiable(litMap.Conflicts(giniSolver))
	}

	// This should never happen
	return nil, errors.New("unknown outcome")
}

func New(options ...Option) (*Solver, error) {
	s := Solver{}
	for _, option := range append(options, defaults...) {
		if err := option(&s); err != nil {
			return nil, err
		}
	}
	return &s, nil
}

type Option func(s *Solver) error

func WithTracer(t deppy.Tracer) Option {
	return func(s *Solver) error {
		s.tracer = t
		return nil
	}
}

var defaults = []Option{
	func(s *Solver) error {
		if s.tracer == nil {
			s.tracer = DefaultTracer{}
		}
		return nil
	},
}
