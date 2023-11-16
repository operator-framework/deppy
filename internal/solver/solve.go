package solver

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-air/gini"
	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/z"

	"github.com/operator-framework/deppy/pkg/deppy"
)

var ErrIncomplete = errors.New("cancelled before a solution could be found")

type Solver interface {
	Solve(context.Context) ([]deppy.Variable, error)
}

type solver struct {
	g      inter.S
	litMap *litMapping
	tracer deppy.Tracer
	buffer []z.Lit
}

const (
	satisfiable   = 1
	unsatisfiable = -1
	unknown       = 0
)

// Solve takes a slice containing all Variables and returns a slice
// containing only those Variables that were selected for
// installation. If no solution is possible, or if the provided
// Context times out or is cancelled, an error is returned.
func (s *solver) Solve(_ context.Context) ([]deppy.Variable, error) {
	result, err := s.solve()

	// This likely indicates a bug, so discard whatever
	// return values were produced.
	if derr := s.litMap.Error(); derr != nil {
		return nil, derr
	}

	return result, err
}

func (s *solver) solve() ([]deppy.Variable, error) {
	// teach all constraints to the solver
	s.litMap.AddConstraints(s.g)

	// collect literals of all mandatory variables to assume as a baseline
	anchors := s.litMap.AnchorIdentifiers()
	assumptions := make([]z.Lit, len(anchors))
	for i := range anchors {
		assumptions[i] = s.litMap.LitOf(anchors[i])
	}

	// assume that all constraints hold
	s.litMap.AssumeConstraints(s.g)
	s.g.Assume(assumptions...)

	var aset map[z.Lit]struct{}
	// push a new test scope with the baseline assumptions, to prevent them from being cleared during search
	outcome, _ := s.g.Test(nil)
	if outcome != satisfiable && outcome != unsatisfiable {
		// searcher for solutions in input Order, so that preferences
		// can be taken into acount (i.e. prefer one catalog to another)
		outcome, assumptions, aset = (&search{s: s.g, lits: s.litMap, tracer: s.tracer}).Do(context.Background(), assumptions)
	}
	switch outcome {
	case satisfiable:
		s.buffer = s.litMap.Lits(s.buffer)
		var extras, excluded []z.Lit
		for _, m := range s.buffer {
			if _, ok := aset[m]; ok {
				continue
			}
			if !s.g.Value(m) {
				excluded = append(excluded, m.Not())
				continue
			}
			extras = append(extras, m)
		}
		s.g.Untest()
		cs := s.litMap.CardinalityConstrainer(s.g, extras)
		s.g.Assume(assumptions...)
		s.g.Assume(excluded...)
		s.litMap.AssumeConstraints(s.g)
		_, s.buffer = s.g.Test(s.buffer)
		for w := 0; w <= cs.N(); w++ {
			s.g.Assume(cs.Leq(w))
			if s.g.Solve() == satisfiable {
				return s.litMap.Variables(s.g), nil
			}
		}
		// Something is wrong if we can't find a model anymore
		// after optimizing for cardinality.
		return nil, fmt.Errorf("unexpected internal error")
	case unsatisfiable:
		return nil, deppy.NotSatisfiable(s.litMap.Conflicts(s.g))
	}

	return nil, ErrIncomplete
}

func NewSolver(options ...Option) (Solver, error) {
	s := solver{g: gini.New()}
	for _, option := range append(options, defaults...) {
		if err := option(&s); err != nil {
			return nil, err
		}
	}
	return &s, nil
}

type Option func(s *solver) error

func WithInput(input []deppy.Variable) Option {
	return func(s *solver) error {
		var err error
		s.litMap, err = newLitMapping(input)
		return err
	}
}

func WithTracer(t deppy.Tracer) Option {
	return func(s *solver) error {
		s.tracer = t
		return nil
	}
}

var defaults = []Option{
	func(s *solver) error {
		if s.litMap == nil {
			var err error
			s.litMap, err = newLitMapping(nil)
			return err
		}
		return nil
	},
	func(s *solver) error {
		if s.tracer == nil {
			s.tracer = DefaultTracer{}
		}
		return nil
	},
}
