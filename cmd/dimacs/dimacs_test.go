package dimacs_test

import (
	"bytes"
	"testing"

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
