package solver_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gstruct"

	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/sat"
	"github.com/operator-framework/deppy/internal/solver"
)

type EntitySourceStruct struct {
	variables []sat.Variable
	entitysource.EntityQuerier
	entitysource.EntityContentGetter
}

func (c EntitySourceStruct) GetVariables(_ context.Context, _ entitysource.EntityQuerier) ([]sat.Variable, error) {
	return c.variables, nil
}

func NewEntitySource(variables []sat.Variable) *EntitySourceStruct {
	entities := make(map[entitysource.EntityID]entitysource.Entity, len(variables))
	for _, variable := range variables {
		entityID := entitysource.EntityID(variable.Identifier())
		entities[entityID] = *entitysource.NewEntity(entityID, map[string]string{"x": "y"})
	}
	return &EntitySourceStruct{
		variables:           variables,
		EntityQuerier:       entitysource.NewCacheQuerier(entities),
		EntityContentGetter: entitysource.NoContentSource(),
	}
}

var _ = Describe("Entity", func() {
	It("should select a mandatory entity", func() {
		variables := []sat.Variable{
			constraints.NewVariable("1", sat.Mandatory()),
			constraints.NewVariable("2"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
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
			constraints.NewVariable("1", sat.Mandatory()),
			constraints.NewVariable("2", sat.Mandatory()),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
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
			constraints.NewVariable("1", sat.Mandatory(), sat.Dependency("2")),
			constraints.NewVariable("2"),
			constraints.NewVariable("3"),
		}
		s := NewEntitySource(variables)

		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
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
			constraints.NewVariable("1", sat.Mandatory(), sat.Dependency("2")),
			constraints.NewVariable("2", sat.Prohibited()),
			constraints.NewVariable("3"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
		Expect(err).To(BeNil())
		_, err = so.Solve(context.Background())
		Expect(err).Should(HaveOccurred())
	})

	It("should select a mandatory entity and its dependency and ignore a non-mandatory prohibited variable", func() {
		variables := []sat.Variable{
			constraints.NewVariable("1", sat.Mandatory(), sat.Dependency("2")),
			constraints.NewVariable("2"),
			constraints.NewVariable("3", sat.Prohibited()),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
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
			constraints.NewVariable("1", sat.Or("2", false, false), sat.Dependency("3")),
			constraints.NewVariable("2", sat.Dependency("4")),
			constraints.NewVariable("3", sat.Prohibited()),
			constraints.NewVariable("4"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
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
			constraints.NewVariable("1", sat.Or("2", false, false), sat.Dependency("3"), sat.Dependency("4")),
			constraints.NewVariable("2", sat.Dependency("3")),
			constraints.NewVariable("3", sat.AtMost(1, "3", "4")),
			constraints.NewVariable("4"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
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
			constraints.NewVariable("1", sat.Or("2", false, false), sat.Dependency("3"), sat.Dependency("4")),
			constraints.NewVariable("2", sat.Dependency("4"), sat.Dependency("5")),
			constraints.NewVariable("3", sat.Conflict("6")),
			constraints.NewVariable("4", sat.Dependency("6")),
			constraints.NewVariable("5"),
			constraints.NewVariable("6"),
		}
		s := NewEntitySource(variables)
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
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
