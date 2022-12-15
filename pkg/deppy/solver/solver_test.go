package solver_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/pkg/deppy/input"

	"github.com/operator-framework/deppy/pkg/deppy/solver"

	"github.com/operator-framework/deppy/pkg/deppy/constraint"

	"github.com/operator-framework/deppy/pkg/deppy"

	. "github.com/onsi/gomega/gstruct"
)

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
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(true),
			deppy.Identifier("2"): Equal(false),
		}))
	})

	It("should select two mandatory entities", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory()),
			input.NewSimpleVariable("2", constraint.Mandatory()),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(true),
			deppy.Identifier("2"): Equal(true),
		}))
	})

	It("should select a mandatory entity and its dependency", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3"),
		}
		s := NewEntitySource(variables)

		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(true),
			deppy.Identifier("2"): Equal(true),
			deppy.Identifier("3"): Equal(false),
		}))
	})

	It("should fail when a dependency is prohibited", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2", constraint.Prohibited()),
			input.NewSimpleVariable("3"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		_, err = so.Solve(context.Background())
		Expect(err).Should(HaveOccurred())
	})

	It("should select a mandatory entity and its dependency and ignore a non-mandatory prohibited variable", func() {
		variables := []deppy.Variable{
			input.NewSimpleVariable("1", constraint.Mandatory(), constraint.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3", constraint.Prohibited()),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(true),
			deppy.Identifier("2"): Equal(true),
			deppy.Identifier("3"): Equal(false),
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
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(false),
			deppy.Identifier("2"): Equal(true),
			deppy.Identifier("3"): Equal(false),
			deppy.Identifier("4"): Equal(true),
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
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(false),
			deppy.Identifier("2"): Equal(true),
			deppy.Identifier("3"): Equal(true),
			deppy.Identifier("4"): Equal(false),
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
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			deppy.Identifier("1"): Equal(false),
			deppy.Identifier("2"): Equal(true),
			deppy.Identifier("3"): Equal(false),
			deppy.Identifier("4"): Equal(true),
			deppy.Identifier("5"): Equal(true),
			deppy.Identifier("6"): Equal(true),
		}))
	})
})
