package input_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/pkg/input"

	. "github.com/onsi/gomega/gstruct"

	"github.com/operator-framework/deppy/pkg/solver"
)

type EntitySourceStruct struct {
	variables []solver.Variable
	input.EntitySource
}

func (c EntitySourceStruct) GetVariables(_ context.Context, _ input.EntitySource) ([]solver.Variable, error) {
	return c.variables, nil
}

func NewEntitySource(variables []solver.Variable) *EntitySourceStruct {
	entities := make(map[solver.Identifier]input.Entity, len(variables))
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
		variables := []solver.Variable{
			input.NewSimpleVariable("1", solver.Mandatory()),
			input.NewSimpleVariable("2"),
		}
		s := NewEntitySource(variables)
		so, err := input.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			solver.Identifier("1"): Equal(true),
			solver.Identifier("2"): Equal(false),
		}))
	})

	It("should select two mandatory entities", func() {
		variables := []solver.Variable{
			input.NewSimpleVariable("1", solver.Mandatory()),
			input.NewSimpleVariable("2", solver.Mandatory()),
		}
		s := NewEntitySource(variables)
		so, err := input.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			solver.Identifier("1"): Equal(true),
			solver.Identifier("2"): Equal(true),
		}))
	})

	It("should select a mandatory entity and its dependency", func() {
		variables := []solver.Variable{
			input.NewSimpleVariable("1", solver.Mandatory(), solver.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3"),
		}
		s := NewEntitySource(variables)

		so, err := input.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			solver.Identifier("1"): Equal(true),
			solver.Identifier("2"): Equal(true),
			solver.Identifier("3"): Equal(false),
		}))
	})

	It("should fail when a dependency is prohibited", func() {
		variables := []solver.Variable{
			input.NewSimpleVariable("1", solver.Mandatory(), solver.Dependency("2")),
			input.NewSimpleVariable("2", solver.Prohibited()),
			input.NewSimpleVariable("3"),
		}
		s := NewEntitySource(variables)
		so, err := input.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		_, err = so.Solve(context.Background())
		Expect(err).Should(HaveOccurred())
	})

	It("should select a mandatory entity and its dependency and ignore a non-mandatory prohibited variable", func() {
		variables := []solver.Variable{
			input.NewSimpleVariable("1", solver.Mandatory(), solver.Dependency("2")),
			input.NewSimpleVariable("2"),
			input.NewSimpleVariable("3", solver.Prohibited()),
		}
		s := NewEntitySource(variables)
		so, err := input.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			solver.Identifier("1"): Equal(true),
			solver.Identifier("2"): Equal(true),
			solver.Identifier("3"): Equal(false),
		}))
	})

	It("should not select 'or' paths that are prohibited", func() {
		variables := []solver.Variable{
			input.NewSimpleVariable("1", solver.Or("2", false, false), solver.Dependency("3")),
			input.NewSimpleVariable("2", solver.Dependency("4")),
			input.NewSimpleVariable("3", solver.Prohibited()),
			input.NewSimpleVariable("4"),
		}
		s := NewEntitySource(variables)
		so, err := input.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			solver.Identifier("1"): Equal(false),
			solver.Identifier("2"): Equal(true),
			solver.Identifier("3"): Equal(false),
			solver.Identifier("4"): Equal(true),
		}))
	})

	It("should respect atMost constraint", func() {
		variables := []solver.Variable{
			input.NewSimpleVariable("1", solver.Or("2", false, false), solver.Dependency("3"), solver.Dependency("4")),
			input.NewSimpleVariable("2", solver.Dependency("3")),
			input.NewSimpleVariable("3", solver.AtMost(1, "3", "4")),
			input.NewSimpleVariable("4"),
		}
		s := NewEntitySource(variables)
		so, err := input.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			solver.Identifier("1"): Equal(false),
			solver.Identifier("2"): Equal(true),
			solver.Identifier("3"): Equal(true),
			solver.Identifier("4"): Equal(false),
		}))
	})

	It("should respect dependency conflicts", func() {
		variables := []solver.Variable{
			input.NewSimpleVariable("1", solver.Or("2", false, false), solver.Dependency("3"), solver.Dependency("4")),
			input.NewSimpleVariable("2", solver.Dependency("4"), solver.Dependency("5")),
			input.NewSimpleVariable("3", solver.Conflict("6")),
			input.NewSimpleVariable("4", solver.Dependency("6")),
			input.NewSimpleVariable("5"),
			input.NewSimpleVariable("6"),
		}
		s := NewEntitySource(variables)
		so, err := input.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			solver.Identifier("1"): Equal(false),
			solver.Identifier("2"): Equal(true),
			solver.Identifier("3"): Equal(false),
			solver.Identifier("4"): Equal(true),
			solver.Identifier("5"): Equal(true),
			solver.Identifier("6"): Equal(true),
		}))
	})
})
