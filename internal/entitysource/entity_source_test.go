package entitysource_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gstruct"

	"github.com/operator-framework/deppy/internal/entitysource"
)

type testentitysource struct {
	entitysource.EntityQuerier
	entitysource.EntityContentGetter
}

// Test functions for filter
func index(index string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		i, err := entity.GetProperty("index")
		if err != nil {
			return false
		}
		if i == index {
			return true
		}
		return false
	}
}
func source(source string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		i, err := entity.GetProperty("entity")
		if err != nil {
			return false
		}
		if i == source {
			return true
		}
		return false
	}
}

// Test function for iterate
var entitycheck map[entitysource.EntityID]bool

func check(entity *entitysource.Entity) error {
	checked, ok := entitycheck[entity.ID()]
	Expect(ok).Should(BeTrue())
	Expect(checked).Should(BeFalse())
	entitycheck[entity.ID()] = true
	return nil
}

// Test function for GroupBy
func sourceindex(entity *entitysource.Entity) []string {
	switch entity.ID() {
	case entitysource.EntityID("1-1"):
		return []string{"source 1", "index 1"}
	case entitysource.EntityID("1-2"):
		return []string{"source 1", "index 2"}
	case entitysource.EntityID("2-1"):
		return []string{"source 2", "index 1"}
	case entitysource.EntityID("2-2"):
		return []string{"source 2", "index 2"}
	}
	return nil
}

var _ = Describe("EntitySource", func() {
	When("a group is created with multiple entity sources", func() {
		var (
			group entitysource.EntitySource
		)
		BeforeEach(func() {

			entities1 := map[entitysource.EntityID]entitysource.Entity{
				entitysource.EntityID("1-1"): *entitysource.NewEntity(entitysource.EntityID("1-1"), map[string]string{"entity": "1", "index": "1"}),
				entitysource.EntityID("1-2"): *entitysource.NewEntity(entitysource.EntityID("1-2"), map[string]string{"entity": "1", "index": "2"}),
			}
			entities2 := map[entitysource.EntityID]entitysource.Entity{
				entitysource.EntityID("2-1"): *entitysource.NewEntity(entitysource.EntityID("2-1"), map[string]string{"entity": "2", "index": "1"}),
				entitysource.EntityID("2-2"): *entitysource.NewEntity(entitysource.EntityID("2-2"), map[string]string{"entity": "2", "index": "2"}),
			}

			entitysource1 := testentitysource{EntityQuerier: entitysource.NewCacheQuerier(entities1), EntityContentGetter: entitysource.NoContentSource()}
			entitysource2 := testentitysource{EntityQuerier: entitysource.NewCacheQuerier(entities2), EntityContentGetter: entitysource.NoContentSource()}
			group = entitysource.NewGroup(entitysource1, entitysource2)
		})

		It("Get should return requested entity", func() {
			e := group.Get(context.Background(), entitysource.EntityID("2-2"))
			Expect(e).NotTo(BeNil())
			Expect(e.ID()).To(Equal(entitysource.EntityID("2-2")))
		})

		It("Filter should return selected entities", func() {
			id := func(element interface{}) string {
				return fmt.Sprintf("%v", element)
			}
			el, err := group.Filter(context.Background(), entitysource.Or(index("2"), source("1")))
			Expect(err).To(BeNil())
			Expect(el).To(MatchAllElements(id, Elements{
				"{1-2 map[entity:1 index:2]}": Not(BeNil()),
				"{2-2 map[entity:2 index:2]}": Not(BeNil()),
				"{1-1 map[entity:1 index:1]}": Not(BeNil()),
			}))
			ids := el.CollectIds()
			Expect(ids).NotTo(BeNil())
			Expect(ids).To(MatchAllElements(id, Elements{
				"1-2": Not(BeNil()),
				"2-2": Not(BeNil()),
				"1-1": Not(BeNil()),
			}))

			el, err = group.Filter(context.Background(), entitysource.And(index("2"), source("1")))
			Expect(err).To(BeNil())
			Expect(el).To(MatchAllElements(id, Elements{
				"{1-2 map[entity:1 index:2]}": Not(BeNil()),
			}))
			ids = el.CollectIds()
			Expect(ids).NotTo(BeNil())
			Expect(ids).To(MatchAllElements(id, Elements{
				"1-2": Not(BeNil()),
			}))

			el, err = group.Filter(context.Background(), entitysource.And(index("2"), entitysource.Not(source("1"))))
			Expect(err).To(BeNil())
			Expect(el).To(MatchAllElements(id, Elements{
				"{2-2 map[entity:2 index:2]}": Not(BeNil()),
			}))
			ids = el.CollectIds()
			Expect(ids).NotTo(BeNil())
			Expect(ids).To(MatchAllElements(id, Elements{
				"2-2": Not(BeNil()),
			}))

		})

		It("Iterate should go through all entities", func() {
			entitycheck = map[entitysource.EntityID]bool{entitysource.EntityID("1-1"): false, entitysource.EntityID("1-2"): false, entitysource.EntityID("2-1"): false, entitysource.EntityID("2-2"): false}
			err := group.Iterate(context.Background(), check)
			Expect(err).To(BeNil())
			for _, value := range entitycheck {
				Expect(value).To(BeTrue())
			}
		})

		It("GroupBy should group entities by the keys", func() {
			id := func(element interface{}) string {
				return fmt.Sprintf("%v", element)
			}
			grouped, err := group.GroupBy(context.Background(), sourceindex)
			Expect(err).To(BeNil())
			Expect(grouped).To(MatchAllKeys(Keys{
				"index 1":  Not(BeNil()),
				"index 2":  Not(BeNil()),
				"source 1": Not(BeNil()),
				"source 2": Not(BeNil()),
			}))
			for key, value := range grouped {
				switch key {
				case "index 1":
					Expect(value).To(MatchAllElements(id, Elements{
						"{1-1 map[entity:1 index:1]}": Not(BeNil()),
						"{2-1 map[entity:2 index:1]}": Not(BeNil()),
					}))
				case "index 2":
					Expect(value).To(MatchAllElements(id, Elements{
						"{1-2 map[entity:1 index:2]}": Not(BeNil()),
						"{2-2 map[entity:2 index:2]}": Not(BeNil()),
					}))
				case "source 1":
					Expect(value).To(MatchAllElements(id, Elements{
						"{1-1 map[entity:1 index:1]}": Not(BeNil()),
						"{1-2 map[entity:1 index:2]}": Not(BeNil()),
					}))
				case "source 2":
					Expect(value).To(MatchAllElements(id, Elements{
						"{2-1 map[entity:2 index:1]}": Not(BeNil()),
						"{2-2 map[entity:2 index:2]}": Not(BeNil()),
					}))
				}

			}
		})
	})
})
