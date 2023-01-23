package constraint_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/pkg/deppy/constraint"

	"github.com/operator-framework/deppy/pkg/deppy"
)

func TestPkg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Constraint Suite")
}

var _ = Describe("Constraint", func() {
	Describe("UserFriendlyConstraint", func() {
		It("should provide the custom constraint message", func() {
			userFriendlyConstraint := constraint.NewUserFriendlyConstraint(constraint.Mandatory(), func(constraint deppy.Constraint, subject deppy.Identifier) string {
				return fmt.Sprintf("'%s' just _has_ to be there or you can't even...", subject)
			})
			Expect(userFriendlyConstraint.String("this thing")).To(Equal("'this thing' just _has_ to be there or you can't even..."))
		})
	})
})
