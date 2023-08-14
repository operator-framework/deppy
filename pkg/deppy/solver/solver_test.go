package solver_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/operator-framework/deppy/pkg/deppy/input"

	"github.com/operator-framework/deppy/pkg/deppy/solver"

	"github.com/operator-framework/deppy/pkg/deppy/constraint"

	"github.com/operator-framework/deppy/pkg/deppy"
)

func TestSolver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Solver Suite")
}

type EntitySourceStruct struct {
	variables []deppy.Variable
	input.EntitySource
}

func (c EntitySourceStruct) GetVariables(_ context.Context, _ input.EntitySource) ([]deppy.Variable, error) {
	return c.variables, nil
}

func NewEntitySource(variables []deppy.Variable) *EntitySourceStruct {
	entities := make(map[deppy.Identifier]input.Entity, len(variables))
	for _, variable := range variables {
		entityID := variable.Identifier()
		entities[entityID] = *input.NewEntity(entityID, map[string]string{"x": "y"})
	}
	return &EntitySourceStruct{
		variables:    variables,
		EntitySource: input.NewCacheQuerier(entities),
	}
}

var _ = Describe("Entity", func() {
	It("should select a mandatory entity", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory()),
			input.NewSimpleVariable("2"),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(input.NewSimpleVariable("1", constraint.Mandatory())),
		}))
		Expect(solution.AllVariables()).To(BeNil())
	})

	It("should select two mandatory entities", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory()),
			input.NewSimpleVariable("2", constraint.Mandatory()),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(input.NewSimpleVariable("1", constraint.Mandatory())),
			deppy.Identifier("2"): Equal(input.NewSimpleVariable("2", constraint.Mandatory())),
		}))
	})

	It("should select a mandatory entity and its dependency", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3"),
		}
		s := NewEntitySource(variables)

		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2"))),
			deppy.Identifier("2"): Equal(input.NewSimpleVariable("2")),
		}))
	})

	It("should return untyped nil error from solution.Error() when there is a solution", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory()),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution).ToNot(BeNil())

		// Using this style for the assertion to ensure that gomega
		// doesn't equate nil errors of different types.
		if err := solution.Error(); err != nil {
			Fail(fmt.Sprintf("expected solution.Error() to be untyped nil, got %#v", solution.Error()))
		}
	})

	It("should place resolution errors in the solution", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2", constraint.Prohibited()),
			input.NewSimpleVariable("3"),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).To(HaveOccurred())
	})

	It("should return peripheral errors", func() {
		s := NewEntitySource(nil)
		so := solver.NewDeppySolver(s, FailingVariableSource{})
		solution, err := so.Solve(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(solution).To(BeNil())
	})

	It("should select a mandatory entity and its dependency and ignore a non-mandatory prohibited variable", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3", constraint.Prohibited()),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2"))),
			deppy.Identifier("2"): Equal(input.NewSimpleVariable("2")),
		}))
	})

	It("should add all variables to solution if option is given", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3", constraint.Prohibited()),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background(), solver.AddAllVariablesToSolution())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).ToNot(HaveOccurred())
		Expect(solution.AllVariables()).To(Equal([]deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3", constraint.Prohibited()),
		}))
	})

	It("should not select 'or' paths that are prohibited", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Or("2", false, false), constraint.Dependency("3")),
			input.NewSimpleVariable("2", constraint.Dependency("4")),
			input.NewSimpleVariable("3", constraint.Prohibited()),
			input.NewSimpleVariable("4"),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("2"): Equal(input.NewSimpleVariable("2", constraint.Dependency("4"))),
			deppy.Identifier("4"): Equal(input.NewSimpleVariable("4")),
		}))
	})

	It("should respect atMost constraint", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Or("2", false, false), constraint.Dependency("3"), constraint.Dependency("4")),
			input.NewSimpleVariable("2", constraint.Dependency("3")),
			input.NewSimpleVariable("3", constraint.AtMost(1, "3", "4")),
			input.NewSimpleVariable("4"),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("2"): Equal(input.NewSimpleVariable("2", constraint.Dependency("3"))),
			deppy.Identifier("3"): Equal(input.NewSimpleVariable("3", constraint.AtMost(1, "3", "4"))),
		}))
	})

	It("should respect dependency conflicts", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Or("2", false, false), constraint.Dependency("3"), constraint.Dependency("4")),
			input.NewSimpleVariable("2", constraint.Dependency("4"), constraint.Dependency("5")),
			input.NewSimpleVariable("3", constraint.Conflict("6")),
			input.NewSimpleVariable("4", constraint.Dependency("6")),
			input.NewSimpleVariable("5"),
			input.NewSimpleVariable("6"),
		}
		s := NewEntitySource(variables)
		so := solver.NewDeppySolver(s, s)
		solution, err := so.Solve(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("2"): Equal(input.NewSimpleVariable("2", constraint.Dependency("4"), constraint.Dependency("5"))),
			deppy.Identifier("4"): Equal(input.NewSimpleVariable("4", constraint.Dependency("6"))),
			deppy.Identifier("5"): Equal(input.NewSimpleVariable("5")),
			deppy.Identifier("6"): Equal(input.NewSimpleVariable("6")),
		}))
	})
})

var _ input.VariableSource = &FailingVariableSource{}

type FailingVariableSource struct {
}

func (f FailingVariableSource) GetVariables(_ context.Context, _ input.EntitySource) ([]deppy.Variable, error) {
	return nil, fmt.Errorf("error")
}
