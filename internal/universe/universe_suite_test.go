package universe_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUniverse(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Universe Suite")
}
