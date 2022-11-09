package solver_test

import (
	"context"
	"strconv"

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

func (c EntitySourceStruct) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	return c.variables, nil
}

func createEntities(num int) map[entitysource.EntityID]entitysource.Entity {
	entities := make(map[entitysource.EntityID]entitysource.Entity, num)
	for i := 0; i < num; i++ {
		//id := entitysource.EntityID(string([]byte{byte('1' + i)}))
		id := entitysource.EntityID(strconv.Itoa(i + 1))
		entities[id] = *entitysource.NewEntity(id, map[string]string{"p1": "x", "p2": "y"})
	}
	return entities
}

var _ = Describe("Entity", func() {
	It("1 mandatory", func() {
		entities := createEntities(2)
		variables := []sat.Variable{
			constraints.NewVariable(sat.Identifier("1"), sat.Mandatory()),
			constraints.NewVariable(sat.Identifier("2")),
		}
		s := &EntitySourceStruct{
			EntityQuerier:       entitysource.NewCacheQuerier(entities),
			EntityContentGetter: entitysource.NoContentSource(),
			variables:           variables,
		}
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(true),
			entitysource.EntityID("2"): Equal(false),
		}))
	})

	It("2 mandatories", func() {
		entities := createEntities(2)
		variables := []sat.Variable{
			constraints.NewVariable(sat.Identifier("1"), sat.Mandatory()),
			constraints.NewVariable(sat.Identifier("2"), sat.Mandatory()),
		}
		s := &EntitySourceStruct{
			EntityQuerier:       entitysource.NewCacheQuerier(entities),
			EntityContentGetter: entitysource.NoContentSource(),
			variables:           variables,
		}
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
		Expect(err).To(BeNil())
		solution, err := so.Solve(context.Background())
		Expect(err).To(BeNil())
		Expect(solution).To(MatchAllKeys(Keys{
			entitysource.EntityID("1"): Equal(true),
			entitysource.EntityID("2"): Equal(true),
		}))
	})

	It("1 mandatory, 1 dependent", func() {
		entities := createEntities(3)
		variables := []sat.Variable{
			constraints.NewVariable(sat.Identifier("1"), sat.Mandatory(), sat.Dependency(sat.Identifier("2"))),
			constraints.NewVariable(sat.Identifier("2")),
			constraints.NewVariable(sat.Identifier("3")),
		}
		s := &EntitySourceStruct{
			EntityQuerier:       entitysource.NewCacheQuerier(entities),
			EntityContentGetter: entitysource.NoContentSource(),
			variables:           variables,
		}
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

	It("1 mandatory, 1 dependent, dependent Prohibited", func() {
		entities := createEntities(3)
		variables := []sat.Variable{
			constraints.NewVariable(sat.Identifier("1"), sat.Mandatory(), sat.Dependency(sat.Identifier("2"))),
			constraints.NewVariable(sat.Identifier("2"), sat.Prohibited()),
			constraints.NewVariable(sat.Identifier("3")),
		}

		s := &EntitySourceStruct{
			EntityQuerier:       entitysource.NewCacheQuerier(entities),
			EntityContentGetter: entitysource.NoContentSource(),
			variables:           variables,
		}
		so, err := solver.NewDeppySolver(entitysource.NewGroup(s), constraints.NewConstraintAggregator(s))
		Expect(err).To(BeNil())
		_, err = so.Solve(context.Background())
		Expect(err).Should(HaveOccurred())
	})

	It("1 mandatory, 1 dependent, none-dependent Prohibited", func() {
		entities := createEntities(3)
		variables := []sat.Variable{
			constraints.NewVariable(sat.Identifier("1"), sat.Mandatory(), sat.Dependency(sat.Identifier("2"))),
			constraints.NewVariable(sat.Identifier("2")),
			constraints.NewVariable(sat.Identifier("3"), sat.Prohibited()),
		}
		s := &EntitySourceStruct{
			EntityQuerier:       entitysource.NewCacheQuerier(entities),
			EntityContentGetter: entitysource.NoContentSource(),
			variables:           variables,
		}
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

	It("or, dependent, dependent Prohibited", func() {
		entities := createEntities(4)
		variables := []sat.Variable{
			constraints.NewVariable(sat.Identifier("1"), sat.Or(sat.Identifier("2"), false, false), sat.Dependency(sat.Identifier("3"))),
			constraints.NewVariable(sat.Identifier("2"), sat.Dependency(sat.Identifier("4"))),
			constraints.NewVariable(sat.Identifier("3"), sat.Prohibited()),
			constraints.NewVariable(sat.Identifier("4")),
		}
		s := &EntitySourceStruct{
			EntityQuerier:       entitysource.NewCacheQuerier(entities),
			EntityContentGetter: entitysource.NoContentSource(),
			variables:           variables,
		}
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

	It("at most, or, dependent", func() {
		entities := createEntities(4)
		variables := []sat.Variable{
			constraints.NewVariable(sat.Identifier("1"), sat.Or(sat.Identifier("2"), false, false), sat.Dependency(sat.Identifier("3")), sat.Dependency(sat.Identifier("4"))),
			constraints.NewVariable(sat.Identifier("2"), sat.Dependency(sat.Identifier("3"))),
			constraints.NewVariable(sat.Identifier("3"), sat.AtMost(1, sat.Identifier("3"), sat.Identifier("4"))),
			constraints.NewVariable(sat.Identifier("4")),
		}
		s := &EntitySourceStruct{
			EntityQuerier:       entitysource.NewCacheQuerier(entities),
			EntityContentGetter: entitysource.NoContentSource(),
			variables:           variables,
		}
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

	It("conflict, or, dependent", func() {
		entities := createEntities(6)
		variables := []sat.Variable{
			constraints.NewVariable(sat.Identifier("1"), sat.Or(sat.Identifier("2"), false, false), sat.Dependency(sat.Identifier("3")), sat.Dependency(sat.Identifier("4"))),
			constraints.NewVariable(sat.Identifier("2"), sat.Dependency(sat.Identifier("4")), sat.Dependency(sat.Identifier("5"))),
			constraints.NewVariable(sat.Identifier("3"), sat.Conflict(sat.Identifier("6"))),
			constraints.NewVariable(sat.Identifier("4"), sat.Dependency(sat.Identifier("6"))),
			constraints.NewVariable(sat.Identifier("5")),
			constraints.NewVariable(sat.Identifier("6")),
		}
		s := &EntitySourceStruct{
			EntityQuerier:       entitysource.NewCacheQuerier(entities),
			EntityContentGetter: entitysource.NoContentSource(),
			variables:           variables,
		}
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
