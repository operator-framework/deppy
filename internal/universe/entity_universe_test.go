package universe_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/internal/source"
	"github.com/operator-framework/deppy/internal/universe"
)

var _ = Describe("EntityUniverse", func() {
	var entityUniverse = universe.NewEntityUniverse([]source.Entity{
		*source.NewEntity("1", map[string]string{"property": "1"}),
		*source.NewEntity("2", map[string]string{"property": "2"}),
		*source.NewEntity("3", map[string]string{"property": "3"}),
	})

	It("should return entities by id", func() {
		entity := entityUniverse.Get("1")
		Expect(entity).NotTo(BeNil())
		Expect(entity).To(Equal(source.NewEntity("1", map[string]string{"property": "1"})))
	})

	It("should return nil if the entity is not found", func() {
		Expect(entityUniverse.Get("not found")).To(BeNil())
	})

	It("should be searchable", func() {
		entities := entityUniverse.Search(func(entity *source.Entity) bool {
			value, err := entity.GetProperty("property")
			if err != nil {
				return false
			}
			return value <= "2"
		})

		Expect(entities).To(Equal(universe.SearchResult([]source.Entity{
			*source.NewEntity("1", map[string]string{"property": "1"}),
			*source.NewEntity("2", map[string]string{"property": "2"}),
		})))
	})

	It("should return all entities", func() {
		Expect(entityUniverse.AllEntities()).To(Equal(universe.SearchResult([]source.Entity{
			*source.NewEntity("1", map[string]string{"property": "1"}),
			*source.NewEntity("2", map[string]string{"property": "2"}),
			*source.NewEntity("3", map[string]string{"property": "3"}),
		})))
	})

})
