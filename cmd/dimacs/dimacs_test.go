package dimacs_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/operator-framework/deppy/pkg/deppy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/cmd/dimacs"
)

func TestDimacs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dimacs Suite")
}

var _ = Describe("Dimacs", func() {
	It("should fail if there is no header", func() {
		problem := "1 2 3 0\n"
		_, err := dimacs.NewDimacs(bytes.NewReader([]byte(problem)))
		Expect(err).To(HaveOccurred())
	})
	It("should fail if there are no clauses", func() {
		problem := "p cnf 3 3\n"
		_, err := dimacs.NewDimacs(bytes.NewReader([]byte(problem)))
		Expect(err).To(HaveOccurred())
	})
	It("should parse valid dimacs", func() {
		problem := "p cnf 3 1\n1 2 3 0\n"
		d, err := dimacs.NewDimacs(bytes.NewReader([]byte(problem)))
		Expect(err).ToNot(HaveOccurred())
		Expect(d.Variables()).To(Equal([]string{"1", "2", "3"}))
		Expect(d.Clauses()).To(Equal([]string{"1 2 3"}))
	})
})

var _ = Describe("Dimacs Variable Source", func() {
	It("should create variables for a dimacs problem", func() {
		problem := "p cnf 3 1\n1 2 3 0\n"
		d, err := dimacs.NewDimacs(bytes.NewReader([]byte(problem)))
		Expect(err).ToNot(HaveOccurred())
		vs := dimacs.NewDimacsVariableSource(d)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		variables, err := vs.GetVariables(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(variables).To(HaveLen(3))

		Expect(variables[0].Identifier()).To(Equal(deppy.Identifier("1")))
		Expect(variables[0].Constraints()).To(HaveLen(1))
		Expect(variables[1].Identifier()).To(Equal(deppy.Identifier("2")))
		Expect(variables[1].Constraints()).To(HaveLen(1))
		Expect(variables[2].Identifier()).To(Equal(deppy.Identifier("3")))
		Expect(variables[2].Constraints()).To(BeEmpty())
	})
})
