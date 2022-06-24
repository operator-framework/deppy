package e2e

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	deppyv1alpha1 "github.com/operator-framework/deppy/api/v1alpha1"
)

const (
	inputClassName = "input-class-name"
)

func Logf(f string, v ...interface{}) {
	if !strings.HasSuffix(f, "\n") {
		f += "\n"
	}
	fmt.Fprintf(GinkgoWriter, f, v...)
}

var _ = Describe("Basic Input resource test", func() {
	When("a valid Input resource is created", func() {
		var (
			input *deppyv1alpha1.Input
			ctx   context.Context
		)
		BeforeEach(func() {
			ctx = context.Background()

			By("creating the testing Input resource")
			input = &deppyv1alpha1.Input{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "input-valid",
				},
				Spec: deppyv1alpha1.InputSpec{
					InputClassName: inputClassName,
					Constraints: []deppyv1alpha1.Constraint{
						{
							Type: "requirePackage",
							Value: map[string]string{
								"Package": "alpha",
								"Version": "0.0.1",
							},
						},
					},
					Properties: []deppyv1alpha1.Property{
						{
							Type: "PropertyType",
							Value: map[string]string{
								"PackageName": "beta",
								"Version":     "0.2.0",
							},
						},
					},
				},
			}
			err := c.Create(ctx, input)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			By("deleting the testing Input resource")
			err := c.Delete(ctx, input)
			Expect(err).To(BeNil())
		})
		It("should consistently contain an empty status", func() {
			Eventually(func() bool {
				if err := c.Get(ctx, client.ObjectKeyFromObject(input), input); err != nil {
					return false
				}
				return len(input.Status.Conditions) == 0
			}).Should(BeTrue())
		})
	})
})
