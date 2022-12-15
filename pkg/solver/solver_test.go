package solver_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gstruct"

	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/sat"
	"github.com/operator-framework/deppy/pkg/solver"
	"github.com/operator-framework/deppy/pkg/variablesource"
)

type EntitySourceStruct struct {
	variables []sat.Variable
	entitysource.EntitySource
}

func (c EntitySourceStruct) GetVariables(_ context.Context, _ entitysource.EntitySource) ([]sat.Variable, error) {
	return c.variables, nil
}

func NewEntitySource(variables []sat.Variable) *EntitySourceStruct {
	entities := make(map[entitysource.EntityID]entitysource.Entity, len(variables))
	for _, variable := range variables {
		entityID := entitysource.EntityID(variable.Identifier())
		entities[entityID] = *entitysource.NewEntity(entityID, map[string]string{"x": "y"})
	}
	return &EntitySourceStruct{
		variables:    variables,
		EntitySource: entitysource.NewCacheQuerier(entities),
	}
}

var _ = Describe("Entity", func() {
	It("should select a mandatory entity", func() {
		variables := []sat.Variable{
			variablesource.NewVariable("1", sat.Mandatory()),
			variablesource.NewVariable("2"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(true),
			entitysource.EntityID("2"): Equal(false),
		}))
	})

	It("should select two mandatory entities", func() {
		variables := []sat.Variable{
			variablesource.NewVariable("1", sat.Mandatory()),
			variablesource.NewVariable("2", sat.Mandatory()),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(true),
			entitysource.EntityID("2"): Equal(true),
		}))
	})

	It("should select a mandatory entity and its dependency", func() {
		variables := []sat.Variable{
			variablesource.NewVariable("1", sat.Mandatory(), sat.Dependency("2")),
			variablesource.NewVariable("2"),
			variablesource.NewVariable("3"),
		}
		s := NewEntitySource(variables)

		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(true),
			entitysource.EntityID("2"): Equal(true),
			entitysource.EntityID("3"): Equal(false),
		}))
	})

	It("should fail when a dependency is prohibited", func() {
		variables := []sat.Variable{
			variablesource.NewVariable("1", sat.Mandatory(), sat.Dependency("2")),
			variablesource.NewVariable("2", sat.Prohibited()),
			variablesource.NewVariable("3"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		_, err = so.Solve(context.Background())
		Expect(err).Should(HaveOccurred())
	})

	It("should select a mandatory entity and its dependency and ignore a non-mandatory prohibited variable", func() {
		variables := []sat.Variable{
			variablesource.NewVariable("1", sat.Mandatory(), sat.Dependency("2")),
			variablesource.NewVariable("2"),
			variablesource.NewVariable("3", sat.Prohibited()),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(true),
			entitysource.EntityID("2"): Equal(true),
			entitysource.EntityID("3"): Equal(false),
		}))
	})

	It("should not select 'or' paths that are prohibited", func() {
		variables := []sat.Variable{
			variablesource.NewVariable("1", sat.Or("2", false, false), sat.Dependency("3")),
			variablesource.NewVariable("2", sat.Dependency("4")),
			variablesource.NewVariable("3", sat.Prohibited()),
			variablesource.NewVariable("4"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(false),
			entitysource.EntityID("2"): Equal(true),
			entitysource.EntityID("3"): Equal(false),
			entitysource.EntityID("4"): Equal(true),
		}))
	})

	It("should respect atMost constraint", func() {
		variables := []sat.Variable{
			variablesource.NewVariable("1", sat.Or("2", false, false), sat.Dependency("3"), sat.Dependency("4")),
			variablesource.NewVariable("2", sat.Dependency("3")),
			variablesource.NewVariable("3", sat.AtMost(1, "3", "4")),
			variablesource.NewVariable("4"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(false),
			entitysource.EntityID("2"): Equal(true),
			entitysource.EntityID("3"): Equal(true),
			entitysource.EntityID("4"): Equal(false),
		}))
	})

	It("should respect dependency conflicts", func() {
		variables := []sat.Variable{
			variablesource.NewVariable("1", sat.Or("2", false, false), sat.Dependency("3"), sat.Dependency("4")),
			variablesource.NewVariable("2", sat.Dependency("4"), sat.Dependency("5")),
			variablesource.NewVariable("3", sat.Conflict("6")),
			variablesource.NewVariable("4", sat.Dependency("6")),
			variablesource.NewVariable("5"),
			variablesource.NewVariable("6"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(s, s)
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(false),
			entitysource.EntityID("2"): Equal(true),
			entitysource.EntityID("3"): Equal(false),
			entitysource.EntityID("4"): Equal(true),
			entitysource.EntityID("5"): Equal(true),
			entitysource.EntityID("6"): Equal(true),
		}))
	})
})
