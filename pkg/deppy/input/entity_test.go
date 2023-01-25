package input_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/pkg/deppy/input/deppyentity"

	"github.com/operator-framework/deppy/pkg/deppy"
)

var _ = Describe("Entity", func() {
	It("stores id and properties", func() {
		entity := deppyentity.NewEntity("id", map[string]string{"prop": "value"})
		Expect(entity.Identifier()).To(Equal(deppy.Identifier("id")))

		value, ok := entity.Properties["prop"]
		Expect(ok).To(BeTrue())
		Expect(value).To(Equal("value"))
	})

	It("returns not found error when property is not found", func() {
		entity := deppyentity.NewEntity("id", map[string]string{"foo": "value"})
		value, ok := entity.Properties["bar"]
		Expect(value).To(Equal(""))
		Expect(ok).To(BeFalse())
	})
})
