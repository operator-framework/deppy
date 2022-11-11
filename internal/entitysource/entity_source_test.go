package entitysource_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gstruct"

	"github.com/operator-framework/deppy/internal/entitysource"
)

type TestEntitySource struct {
	entitysource.EntityQuerier
	entitysource.EntityContentGetter
}

func NewTestEntitySource(entities map[entitysource.EntityID]entitysource.Entity) *TestEntitySource {
	return &TestEntitySource{
		EntityQuerier:       entitysource.NewCacheQuerier(entities),
		EntityContentGetter: entitysource.NoContentSource(),
	}
}

// Test functions for filter
func byIndex(index string) entitysource.Predicate {
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
func bySource(source string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		i, err := entity.GetProperty("source")
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
var entityCheck map[entitysource.EntityID]bool

func check(entity *entitysource.Entity) error {
	checked, ok := entityCheck[entity.ID()]
	Expect(ok).Should(BeTrue())
	Expect(checked).Should(BeFalse())
	entityCheck[entity.ID()] = true
	return nil
}

// Test function for GroupBy
func bySourceAndIndex(entity *entitysource.Entity) []string {
	switch entity.ID() {
	case "1-1":
		return []string{"source 1", "index 1"}
	case "1-2":
		return []string{"source 1", "index 2"}
	case "2-1":
		return []string{"source 2", "index 1"}
	case "2-2":
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
				entitysource.EntityID("1-1"): *entitysource.NewEntity("1-1", map[string]string{"source": "1", "index": "1"}),
				entitysource.EntityID("1-2"): *entitysource.NewEntity("1-2", map[string]string{"source": "1", "index": "2"}),
			}
			entities2 := map[entitysource.EntityID]entitysource.Entity{
				entitysource.EntityID("2-1"): *entitysource.NewEntity("2-1", map[string]string{"source": "2", "index": "1"}),
				entitysource.EntityID("2-2"): *entitysource.NewEntity("2-2", map[string]string{"source": "2", "index": "2"}),
			}
			group = entitysource.NewGroup(NewTestEntitySource(entities1), NewTestEntitySource(entities2))
		})

		Describe("Get", func() {
			It("should return requested entity", func() {
				e := group.Get(context.Background(), "2-2")
				Expect(e).NotTo(BeNil())
				Expect(e.ID()).To(Equal(entitysource.EntityID("2-2")))
			})
		})

		Describe("Filter", func() {
			It("should return entities that meet filter predicates", func() {
				id := func(element interface{}) string {
					return fmt.Sprintf("%v", element)
				}
				el, err := group.Filter(context.Background(), entitysource.Or(byIndex("2"), bySource("1")))
				Expect(err).To(BeNil())
				Expect(el).To(MatchAllElements(id, Elements{
					"{1-2 map[index:2 source:1]}": Not(BeNil()),
					"{2-2 map[index:2 source:2]}": Not(BeNil()),
					"{1-1 map[index:1 source:1]}": Not(BeNil()),
				}))
				ids := el.CollectIds()
				Expect(ids).NotTo(BeNil())
				Expect(ids).To(MatchAllElements(id, Elements{
					"1-2": Not(BeNil()),
					"2-2": Not(BeNil()),
					"1-1": Not(BeNil()),
				}))

				el, err = group.Filter(context.Background(), entitysource.And(byIndex("2"), bySource("1")))
				Expect(err).To(BeNil())
				Expect(el).To(MatchAllElements(id, Elements{
					"{1-2 map[index:2 source:1]}": Not(BeNil()),
				}))
				ids = el.CollectIds()
				Expect(ids).NotTo(BeNil())
				Expect(ids).To(MatchAllElements(id, Elements{
					"1-2": Not(BeNil()),
				}))

				el, err = group.Filter(context.Background(), entitysource.And(byIndex("2"), entitysource.Not(bySource("1"))))
				Expect(err).To(BeNil())
				Expect(el).To(MatchAllElements(id, Elements{
					"{2-2 map[index:2 source:2]}": Not(BeNil()),
				}))
				ids = el.CollectIds()
				Expect(ids).NotTo(BeNil())
				Expect(ids).To(MatchAllElements(id, Elements{
					"2-2": Not(BeNil()),
				}))

			})
		})

		Describe("Iterate", func() {
			It("should go through all entities", func() {
				entityCheck = map[entitysource.EntityID]bool{"1-1": false, "1-2": false, "2-1": false, "2-2": false}
				err := group.Iterate(context.Background(), check)
				Expect(err).To(BeNil())
				for _, value := range entityCheck {
					Expect(value).To(BeTrue())
				}
			})
		})

		Describe("GroupBy", func() {
			It("should group entities by the keys provided by the groupBy function", func() {
				id := func(element interface{}) string {
					return fmt.Sprintf("%v", element)
				}
				grouped, err := group.GroupBy(context.Background(), bySourceAndIndex)
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
							"{1-1 map[index:1 source:1]}": Not(BeNil()),
							"{2-1 map[index:1 source:2]}": Not(BeNil()),
						}))
					case "index 2":
						Expect(value).To(MatchAllElements(id, Elements{
							"{1-2 map[index:2 source:1]}": Not(BeNil()),
							"{2-2 map[index:2 source:2]}": Not(BeNil()),
						}))
					case "source 1":
						Expect(value).To(MatchAllElements(id, Elements{
							"{1-1 map[index:1 source:1]}": Not(BeNil()),
							"{1-2 map[index:2 source:1]}": Not(BeNil()),
						}))
					case "source 2":
						Expect(value).To(MatchAllElements(id, Elements{
							"{2-1 map[index:1 source:2]}": Not(BeNil()),
							"{2-2 map[index:2 source:2]}": Not(BeNil()),
						}))
					default:
						Fail(fmt.Sprintf("unknown key %s", key))
					}
				}
			})
		})
	})
})
