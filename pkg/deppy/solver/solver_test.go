package solver_test

import (
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

var _ = Describe("Entity", func() {
	It("should select a mandatory entity", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory()),
			input.NewSimpleVariable("2"),
		}

		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(input.NewSimpleVariable("1", constraint.Mandatory())),
		}))
	})

	It("should select two mandatory entities", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory()),
			input.NewSimpleVariable("2", constraint.Mandatory()),
		}

		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
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

		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
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
		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
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

		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).To(HaveOccurred())
	})

	It("should select a mandatory entity and its dependency and ignore a non-mandatory prohibited variable", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3", constraint.Prohibited()),
		}

		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
		Expect(err).ToNot(HaveOccurred())
		Expect(solution.Error()).ToNot(HaveOccurred())
		Expect(solution.SelectedVariables()).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2"))),
			deppy.Identifier("2"): Equal(input.NewSimpleVariable("2")),
		}))
	})

	It("should not select 'or' paths that are prohibited", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Or("2", false, false), constraint.Dependency("3")),
			input.NewSimpleVariable("2", constraint.Dependency("4")),
			input.NewSimpleVariable("3", constraint.Prohibited()),
			input.NewSimpleVariable("4"),
		}

		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
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

		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
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

		so := solver.NewDeppySolver()
		solution, err := so.Solve(variables)
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
