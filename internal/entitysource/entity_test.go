package entitysource_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/pkg/entitysource"
)

var _ = Describe("Entity", func() {
	It("stores id and properties", func() {
		entity := entitysource.NewEntity("id", map[string]string{"prop": "value"})
		Expect(entity.ID()).To(Equal(entitysource.EntityID("id")))

		value, err := entity.GetProperty("prop")
		Expect(err).To(BeNil())
		Expect(value).To(Equal("value"))
	})

	It("returns not found error when property is not found", func() {
		entity := entitysource.NewEntity("id", map[string]string{"foo": "value"})
		value, err := entity.GetProperty("bar")
		Expect(value).To(Equal(""))
		Expect(err).To(MatchError(entitysource.EntityPropertyNotFoundError("bar")))
	})
})
