package input_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/pkg/deppy/input"

	"github.com/operator-framework/deppy/pkg/deppy"

	. "github.com/onsi/gomega/gstruct"
)

func TestInput(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Input Suite")
}

// Test functions for filter
func byIndex(index string) input.Predicate {
	return func(entity *input.Entity) bool {
		i, ok := entity.Properties["index"]
		if !ok {
			return false
		}
		if i == index {
			return true
		}
		return false
	}
}
func bySource(source string) input.Predicate {
	return func(entity *input.Entity) bool {
		i, ok := entity.Properties["source"]
		if !ok {
			return false
		}
		if i == source {
			return true
		}
		return false
	}
}

// Test function for iterate
var entityCheck map[deppy.Identifier]bool

func check(entity *input.Entity) error {
	checked, ok := entityCheck[entity.Identifier()]
	Expect(ok).Should(BeTrue())
	Expect(checked).Should(BeFalse())
	entityCheck[entity.Identifier()] = true
	return nil
}

// Test function for GroupBy
func bySourceAndIndex(entity *input.Entity) []string {
	switch entity.Identifier() {
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
			entitySource input.EntitySource
		)

		BeforeEach(func() {
			entities := map[deppy.Identifier]input.Entity{
				deppy.Identifier("1-1"): *input.NewEntity("1-1", map[string]string{"source": "1", "index": "1"}),
				deppy.Identifier("1-2"): *input.NewEntity("1-2", map[string]string{"source": "1", "index": "2"}),
				deppy.Identifier("2-1"): *input.NewEntity("2-1", map[string]string{"source": "2", "index": "1"}),
				deppy.Identifier("2-2"): *input.NewEntity("2-2", map[string]string{"source": "2", "index": "2"}),
			}
			entitySource = input.NewCacheQuerier(entities)
		})

		Describe("Get", func() {
			It("should return requested entity", func() {
				e, err := entitySource.Get(context.Background(), "2-2")
				Expect(err).ToNot(HaveOccurred())
				Expect(e).NotTo(BeNil())
				Expect(e.Identifier()).To(Equal(deppy.Identifier("2-2")))
			})

			It("should return an error when the requested entity is not found", func() {
				e, err := entitySource.Get(context.Background(), "random")
				Expect(err).To(HaveOccurred())
				Expect(e).To(BeNil())
				Expect(err.Error()).To(BeEquivalentTo(fmt.Sprintf("entity with id: %s not found in the entity source", "random")))
			})
		})

		Describe("Filter", func() {
			It("should return entities that meet filter predicates", func() {
				id := func(element interface{}) string {
					return fmt.Sprintf("%v", element)
				}
				el, err := entitySource.Filter(context.Background(), input.Or(byIndex("2"), bySource("1")))
				Expect(err).ToNot(HaveOccurred())
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

				el, err = entitySource.Filter(context.Background(), input.And(byIndex("2"), bySource("1")))
				Expect(err).ToNot(HaveOccurred())
				Expect(el).To(MatchAllElements(id, Elements{
					"{1-2 map[index:2 source:1]}": Not(BeNil()),
				}))
				ids = el.CollectIds()
				Expect(ids).NotTo(BeNil())
				Expect(ids).To(MatchAllElements(id, Elements{
					"1-2": Not(BeNil()),
				}))

				el, err = entitySource.Filter(context.Background(), input.And(byIndex("2"), input.Not(bySource("1"))))
				Expect(err).ToNot(HaveOccurred())
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
				entityCheck = map[deppy.Identifier]bool{"1-1": false, "1-2": false, "2-1": false, "2-2": false}
				err := entitySource.Iterate(context.Background(), check)
				Expect(err).ToNot(HaveOccurred())
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
				grouped, err := entitySource.GroupBy(context.Background(), bySourceAndIndex)
				Expect(err).ToNot(HaveOccurred())
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
