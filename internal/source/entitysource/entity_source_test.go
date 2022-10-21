package entitysource_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"strconv"

	"github.com/operator-framework/deppy/internal/source"
	"github.com/operator-framework/deppy/internal/source/entitysource"
	"github.com/operator-framework/deppy/internal/universe"
)

var entities = []source.Entity{
	*source.NewEntity("1", map[string]string{"property": "1"}),
	*source.NewEntity("2", map[string]string{"property": "2"}),
	*source.NewEntity("3", map[string]string{"property": "3"}),
}

type DeppySource struct {
}

func (ds DeppySource) Sync(ctx context.Context) (*universe.EntityUniverse, error) {
	return universe.NewEntityUniverse(entities), nil
}

// Filter function
func ff2(entity *source.Entity) bool {
	return entity.ID() == "2"
}
func ff13(entity *source.Entity) bool {
	return entity.ID() != "2"
}

// GroupBy function
func gb(entity *source.Entity) []string {
	switch entity.ID() {
	case "1":
		return []string{"odd", "<=2"}
	case "2":
		return []string{"<=2", ">=2"}
	case "3":
		return []string{"odd", ">=2"}
	}
	return []string{}
}

var i map[string]string

// Iterate function
func f(entity *source.Entity) error {
	i[string(entity.ID())] = string(entity.ID())
	return nil
}

var adapter DeppySource

var _ = Describe("EntitySource", func() {
	var ctx = context.Background()

	It("should return new entitySource", func() {
		entities, _ := entitysource.NewEntitySource(ctx, adapter)
		Expect(entities.Get(ctx, "1")).To(Equal(source.NewEntity("1", map[string]string{"property": "1"})))
		Expect(entities.Get(ctx, "2")).To(Equal(source.NewEntity("2", map[string]string{"property": "2"})))
		Expect(entities.Get(ctx, "3")).To(Equal(source.NewEntity("3", map[string]string{"property": "3"})))
	})

	It("should return filtered entities", func() {
		entitiySource, _ := entitysource.NewEntitySource(ctx, adapter)
		entityUniverse, _ := entitiySource.Filter(ctx, ff2)
		entity := entityUniverse.Get("1")
		Expect(entity).To(BeNil())
		entity = entityUniverse.Get("2")
		Expect(entity).To(Equal(source.NewEntity("2", map[string]string{"property": "2"})))
		entity = entityUniverse.Get("3")
		Expect(entity).To(BeNil())

		entityUniverse, _ = entitiySource.Filter(ctx, ff13)
		entity = entityUniverse.Get("1")
		Expect(entity).To(Equal(source.NewEntity("1", map[string]string{"property": "1"})))
		entity = entityUniverse.Get("2")
		Expect(entity).To(BeNil())
		entity = entityUniverse.Get("3")
		Expect(entity).To(Equal(source.NewEntity("3", map[string]string{"property": "3"})))
	})

	It("should return grouped entities", func() {
		entitiySource, _ := entitysource.NewEntitySource(ctx, adapter)
		entityUniverses, _ := entitiySource.GroupBy(ctx, gb)
		entityUniverse := entityUniverses["odd"]
		entity := entityUniverse.Get("1")
		Expect(entity).To(Equal(source.NewEntity("1", map[string]string{"property": "1"})))
		entity = entityUniverse.Get("2")
		Expect(entity).To(BeNil())
		entity = entityUniverse.Get("3")
		Expect(entity).To(Equal(source.NewEntity("3", map[string]string{"property": "3"})))

		entityUniverse = entityUniverses["<=2"]
		entity = entityUniverse.Get("1")
		Expect(entity).To(Equal(source.NewEntity("1", map[string]string{"property": "1"})))
		entity = entityUniverse.Get("2")
		Expect(entity).To(Equal(source.NewEntity("2", map[string]string{"property": "2"})))
		entity = entityUniverse.Get("3")
		Expect(entity).To(BeNil())

		entityUniverse = entityUniverses[">=2"]
		entity = entityUniverse.Get("1")
		Expect(entity).To(BeNil())
		entity = entityUniverse.Get("2")
		Expect(entity).To(Equal(source.NewEntity("2", map[string]string{"property": "2"})))
		entity = entityUniverse.Get("3")
		Expect(entity).To(Equal(source.NewEntity("3", map[string]string{"property": "3"})))
	})

	It("should iterate all entities", func() {
		entitiySource, _ := entitysource.NewEntitySource(ctx, adapter)
		i = make(map[string]string)
		err := entitiySource.Iterate(ctx, f)
		Expect(err).To(BeNil())
		sum := 6
		for key := range i {
			value, err := strconv.Atoi(key)
			Expect(err).To(BeNil())
			sum -= value
		}
		Expect(sum).To(BeZero())
	})
})
