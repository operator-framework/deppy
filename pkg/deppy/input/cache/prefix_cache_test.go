package cache_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/deppy/pkg/deppy/input/cache"
)

func TestDeppy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PrefixCache Suite")
}

var _ = Describe("PrefixCache", func() {
	var c *cache.PrefixCache[string]

	BeforeEach(func() {
		c = cache.NewPrefixCache[string]()
	})

	AfterEach(func() {
		c = nil
	})

	Describe("Get", func() {
		It("should return false if the key is not found", func() {
			value, found := c.Get("a/b/c")
			Expect(found).To(BeFalse())
			Expect(value).To(BeEmpty())
		})

		It("should return true and the value if the key is found", func() {
			c.Set("a/b/c", "value")
			value, found := c.Get("a/b/c")
			Expect(found).To(BeTrue())
			Expect(value).To(Equal("value"))
		})
	})

	Describe("Set", func() {
		It("should set the value for the key", func() {
			c.Set("a/b/c", "value")
			value, found := c.Get("a/b/c")
			Expect(found).To(BeTrue())
			Expect(value).To(Equal("value"))
		})
	})

	Describe("Delete", func() {
		It("should not fail if the key is not found", func() {
			c.Delete("a/b/c")
		})

		It("should delete the value for the key", func() {
			c.Set("a/b/c", "value")
			c.Delete("a/b/c")
			value, found := c.Get("a/b/c")
			Expect(found).To(BeFalse())
			Expect(value).To(BeEmpty())
		})
	})

	Describe("Iterate", func() {
		It("should iterate over all key-value pairs in the cache", func() {
			c.Set("a/b/c", "value1")
			c.Set("a/b/d", "value2")
			c.Set("a/b/e", "value3")

			var result []string
			err := c.Iterate(func(key cache.Key, value string) error {
				result = append(result, string(key)+":"+value)
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf("a/b/c:value1", "a/b/d:value2", "a/b/e:value3"))
		})
	})

	Describe("PrefixScan", func() {
		It("should return all values with matching prefix", func() {
			c.Set("a/b/c", "value1")
			c.Set("a/b/d", "value2")
			c.Set("a/c/e", "value3")

			var result []string
			err := c.PrefixScan("a/b/*", func(key cache.Key, value string) error {
				result = append(result, string(key)+":"+value)
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf("a/b/c:value1", "a/b/d:value2"))
		})

		It("should handle wildcard in prefix", func() {
			c.Set("a/b/c", "value1")
			c.Set("a/b/d", "value2")
			c.Set("a/c/e", "value3")

			var result []string
			err := c.PrefixScan("a/*/e", func(key cache.Key, value string) error {
				result = append(result, string(key)+":"+value)
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf("a/c/e:value3"))
		})

		It("should handle short prefixes as wildcards", func() {
			c.Set("a/b/c", "value1")
			c.Set("a/b/d", "value2")
			c.Set("a/c/e", "value3")

			var result []string
			err := c.PrefixScan("a", func(key cache.Key, value string) error {
				result = append(result, string(key)+":"+value)
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf("a/b/c:value1", "a/b/d:value2", "a/c/e:value3"))
		})
	})

	Describe("DeletePrefix", func() {
		It("should delete values with matching prefix", func() {
			c.Set("a/b/c", "value1")
			c.Set("a/b/d", "value2")
			c.Set("a/c/e", "value3")

			var result []string
			c.DeleteByPrefix("a/b")
			err := c.PrefixScan("a", func(key cache.Key, value string) error {
				result = append(result, string(key)+":"+value)
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf("a/c/e:value3"))
		})

		It("should handle wildcard in prefix", func() {
			c.Set("a/b/c", "value1")
			c.Set("a/b/d", "value2")
			c.Set("a/b/e", "value2")
			c.Set("a/c/e", "value3")

			var result []string
			c.DeleteByPrefix("a/*/e")
			err := c.PrefixScan("a", func(key cache.Key, value string) error {
				result = append(result, string(key)+":"+value)
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf("a/b/c:value1", "a/b/d:value2"))
		})

		It("should handle short prefixes as wildcards", func() {
			c.Set("a/b/c", "value1")
			c.Set("a/b/d", "value2")
			c.Set("a/c/e", "value3")

			var result []string
			c.DeleteByPrefix("a")
			err := c.PrefixScan("a", func(key cache.Key, value string) error {
				result = append(result, string(key)+":"+value)
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})
})
