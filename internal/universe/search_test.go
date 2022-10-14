package universe_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/internal/source"
	"github.com/operator-framework/deppy/internal/universe"
)

var _ = Describe("Search", func() {
	Describe("And Predicate", func() {
		It("should succeed if all predicates are true", func() {
			entity := source.NewEntity("id", map[string]string{})
			predicateOne := func(entity *source.Entity) bool {
				return true
			}
			predicateTwo := func(entity *source.Entity) bool {
				return true
			}

			Expect(universe.And(predicateOne)(entity)).To(BeTrue())
			Expect(universe.And(predicateOne, predicateTwo)(entity)).To(BeTrue())
		})

		It("should succeed if there are no predicates", func() {
			entity := source.NewEntity("id", map[string]string{})
			Expect(universe.And()(entity)).To(BeTrue())
		})

		It("should fail if one predicate is false", func() {
			entity := source.NewEntity("id", map[string]string{})
			predicateOne := func(entity *source.Entity) bool {
				return false
			}
			predicateTwo := func(entity *source.Entity) bool {
				return true
			}

			Expect(universe.And(predicateOne)(entity)).To(BeFalse())
			Expect(universe.And(predicateOne, predicateTwo)(entity)).To(BeFalse())
		})
	})

	Describe("Or Predicate", func() {
		It("should succeed if at least one predicates is true", func() {
			entity := source.NewEntity("id", map[string]string{})
			predicateOne := func(entity *source.Entity) bool {
				return true
			}
			predicateTwo := func(entity *source.Entity) bool {
				return false
			}

			Expect(universe.Or(predicateOne)(entity)).To(BeTrue())
			Expect(universe.Or(predicateOne, predicateTwo)(entity)).To(BeTrue())
		})

		It("should fail if there are no predicates", func() {
			entity := source.NewEntity("id", map[string]string{})
			Expect(universe.Or()(entity)).To(BeFalse())
		})

		It("should fail if all predicates are false", func() {
			entity := source.NewEntity("id", map[string]string{})
			predicateOne := func(entity *source.Entity) bool {
				return false
			}
			predicateTwo := func(entity *source.Entity) bool {
				return false
			}

			Expect(universe.And(predicateOne)(entity)).To(BeFalse())
			Expect(universe.And(predicateOne, predicateTwo)(entity)).To(BeFalse())
		})
	})

	Describe("Not Predicate", func() {
		It("should negate a predicate", func() {
			entity := source.NewEntity("id", map[string]string{})
			predicateOne := func(entity *source.Entity) bool {
				return false
			}
			predicateTwo := func(entity *source.Entity) bool {
				return true
			}

			Expect(universe.Not(predicateOne)(entity)).To(BeTrue())
			Expect(universe.Not(predicateTwo)(entity)).To(BeFalse())
		})
	})

	Describe("SearchResult", func() {
		It("should be sortable", func() {
			results := universe.SearchResult([]source.Entity{
				*source.NewEntity("c", map[string]string{}),
				*source.NewEntity("a", map[string]string{}),
				*source.NewEntity("b", map[string]string{}),
			})
			sortedResults := results.Sort(func(e1 *source.Entity, e2 *source.Entity) bool {
				return e1.ID() < e2.ID()
			})
			Expect(sortedResults).To(Equal(universe.SearchResult([]source.Entity{
				*source.NewEntity("a", map[string]string{}),
				*source.NewEntity("b", map[string]string{}),
				*source.NewEntity("c", map[string]string{}),
			})))
		})

		It("should collect entity ids", func() {
			results := universe.SearchResult([]source.Entity{
				*source.NewEntity("c", map[string]string{}),
				*source.NewEntity("a", map[string]string{}),
				*source.NewEntity("b", map[string]string{}),
			})
			ids := results.CollectIds()
			Expect(ids).To(Equal([]source.EntityID{"c", "a", "b"}))
		})
	})
})
