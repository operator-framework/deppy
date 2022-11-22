package olm_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gstruct"

	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/ext/olm"
)

func TestConstraints(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Constraints Suite")
}

const (
	propertyOLMGVK            = "olm.gvk"
	propertyOLMPackageName    = "olm.packageName"
	propertyOLMVersion        = "olm.version"
	propertyOLMChannel        = "olm.channel"
	propertyOLMDefaultChannel = "olm.defaultChannel"
)

func defaultTestEntityList() entitysource.EntityList {
	return entitysource.EntityList{
		*entitysource.NewEntity("cool-package-1-entity", map[string]string{
			propertyOLMPackageName: "cool-package-1",
			propertyOLMVersion:     "2.0.1",
			propertyOLMChannel:     "channel-1",
			propertyOLMGVK:         "{\"group\":\"my-group\",\"version\":\"my-version\",\"kind\":\"my-kind\"}",
		}),
		*entitysource.NewEntity("cool-package-2-0-entity", map[string]string{
			propertyOLMPackageName: "cool-package-2",
			propertyOLMVersion:     "2.0.3",
			propertyOLMChannel:     "channel-1",
			propertyOLMGVK:         "{\"group\":\"my-group\",\"version\":\"my-version\",\"kind\":\"my-kind\"}",
		}),
		*entitysource.NewEntity("cool-package-2-1-entity", map[string]string{
			propertyOLMPackageName: "cool-package-2",
			propertyOLMVersion:     "2.1.0",
			propertyOLMChannel:     "channel-1",
			propertyOLMGVK:         "{\"group\":\"my-other-group\",\"version\":\"my-version\",\"kind\":\"my-kind\"}",
		}),
		*entitysource.NewEntity("cool-package-3-entity", map[string]string{
			propertyOLMPackageName: "cool-package-3",
			propertyOLMVersion:     "3.1.2",
			propertyOLMChannel:     "channel-2",
			propertyOLMGVK:         "{\"group\":\"my-group\",\"version\":\"my-version\",\"kind\":\"my-kind\"}",
		}),
	}
}

// MockQuerier type to mock the entity querier
type MockQuerier struct {
	testError      error
	testEntityList entitysource.EntityList
}

func (t MockQuerier) Get(ctx context.Context, id entitysource.EntityID) *entitysource.Entity {
	return &entitysource.Entity{}
}
func (t MockQuerier) Filter(ctx context.Context, filter entitysource.Predicate) (entitysource.EntityList, error) {
	if t.testError != nil {
		return nil, t.testError
	}
	ret := entitysource.EntityList{}
	for _, entity := range t.testEntityList {
		if filter(&entity) {
			ret = append(ret, entity)
		}
	}
	return ret, nil
}
func (t MockQuerier) GroupBy(ctx context.Context, id entitysource.GroupByFunction) (entitysource.EntityListMap, error) {
	if t.testError != nil {
		return nil, t.testError
	}
	ret := entitysource.EntityListMap{}
	for _, entity := range t.testEntityList {
		keys := id(&entity)
		for _, key := range keys {
			if _, ok := ret[key]; !ok {
				ret[key] = entitysource.EntityList{}
			}
			ret[key] = append(ret[key], entity)
		}
	}
	return ret, nil
}
func (t MockQuerier) Iterate(ctx context.Context, id entitysource.IteratorFunction) error {
	if t.testError != nil {
		return t.testError
	}
	return nil
}

var _ = Describe("Constraints", func() {
	Context("requirePackage", func() {
		Describe("GetVariables", func() {
			var (
				ctx         context.Context
				mockQuerier MockQuerier
			)
			BeforeEach(func() {
				ctx = context.Background()
				mockQuerier = MockQuerier{
					testError:      nil,
					testEntityList: defaultTestEntityList(),
				}
			})
			// match all
			It("returns one satVar entry describing the required package", func() {
				satVars, err := olm.RequirePackage("cool-package-1", "<=2.0.2", "channel-1").GetVariables(ctx, mockQuerier)
				expectedIdentifier := fmt.Sprintf("require-%s-%s-%s", "cool-package-1", "<=2.0.2", "channel-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Identifier().String()).To(Equal(expectedIdentifier))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[0].String("test-pkg")).To(Equal("test-pkg is mandatory"))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg requires at least one of cool-package-1-entity"))
			})
			// package name
			It("finds no candidates to satisfy the dependency when package name does not match any entities", func() {
				satVars, err := olm.RequirePackage("cool-package-4", "<3.0.0", "channel-1").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			It("finds no candidates to satisfy the dependency when no entries contain the 'olm.packageName' key", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-3-entity", map[string]string{
						"wrong-key":        "cool-package-1",
						propertyOLMVersion: "2.1.2",
						propertyOLMChannel: "channel-1",
					}),
				}
				satVars, err := olm.RequirePackage("cool-package-4", "<=3.0.0", "channel-3").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			// version range
			It("finds no candidates to satisfy the dependency when no entries match the provided version range", func() {
				satVars, err := olm.RequirePackage("cool-package-1", "<=2.0.0", "channel-1").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			It("finds no candidates to satisfy the dependency when given an invalid version range", func() {
				satVars, err := olm.RequirePackage("cool-package-1", "abcdefg", "channel-1").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			It("finds no candidates to satisfy the dependency when no entries have a valid semver value", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-1-entity", map[string]string{
						propertyOLMPackageName: "cool-package-1",
						propertyOLMVersion:     "abcdefg",
						propertyOLMChannel:     "channel-1",
					}),
				}
				satVars, err := olm.RequirePackage("cool-package-1", ">=3.0.0", "channel-1").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			It("finds no candidates to satisfy the dependency when no entries contain the 'olm.version' key", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-1-entity", map[string]string{
						propertyOLMPackageName: "cool-package-1",
						"wrong-key":            "2.1.2",
						propertyOLMChannel:     "channel-1",
					}),
				}
				satVars, err := olm.RequirePackage("cool-package-1", "<=3.0.0", "channel-3").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			// channel
			It("returns one satVar entry describing no possible dependency candidates when the entry has an empty channel name", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-1-entity", map[string]string{
						propertyOLMPackageName: "cool-package-1",
						propertyOLMVersion:     "2.1.2",
						propertyOLMChannel:     "",
					}),
				}
				satVars, err := olm.RequirePackage("cool-package-1", "<=3.0.0", "channel-3").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			It("returns one satVar entry describing dependency candidate when channel requirement is empty", func() {
				satVars, err := olm.RequirePackage("cool-package-1", "<=3.0.0", "").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg requires at least one of cool-package-1-entity"))
			})
			It("returns one satVar entry describing no possible dependency candidates when no entries match the provided channel", func() {
				satVars, err := olm.RequirePackage("cool-package-1", "<=3.0.0", "channel-3").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			It("returns one satVar entry describing no possible dependency candidates when no entries contain the 'olm.channel' key", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-1-entity", map[string]string{
						propertyOLMPackageName: "cool-package-1",
						propertyOLMVersion:     "2.1.2",
						"wrong-key":            "channel-1",
					}),
				}
				satVars, err := olm.RequirePackage("cool-package-1", "<=3.0.0", "channel-1").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Constraints()).Should(HaveLen(2))
				Expect(satVars[0].Constraints()[1].String("test-pkg")).To(Equal("test-pkg has a dependency without any candidates to satisfy it"))
			})
			// entity querier error
			It("forwards any error encountered by the entity querier", func() {
				mockQuerier.testError = errors.New("oh no!")
				satVars, err := olm.RequirePackage("cool-package-1", "<=3.0.0", "channel-1").GetVariables(ctx, mockQuerier)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("oh no!"))
				Expect(satVars).Should(HaveLen(0))
			})
		})
	})
	Context("uniqueness", func() {
		Describe("PackageUniqueness", func() {
			var (
				ctx         context.Context
				mockQuerier MockQuerier
			)
			BeforeEach(func() {
				ctx = context.Background()
				mockQuerier = MockQuerier{
					testError:      nil,
					testEntityList: defaultTestEntityList(),
				}
			})
			It("returns a slice of sat.Variable grouped by package name", func() {
				satVars, err := olm.PackageUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(3))
				sort.Slice(satVars, func(i, j int) bool {
					return satVars[i].Identifier().String() < satVars[j].Identifier().String()
				})
				Expect(satVars[0].Identifier().String()).To(Equal("cool-package-1 uniqueness"))
				Expect(satVars[0].Constraints()[0].String("test-pkg")).To(Equal("test-pkg permits at most 1 of cool-package-1-entity"))
				Expect(satVars[1].Identifier().String()).To(Equal("cool-package-2 uniqueness"))
				Expect(satVars[1].Constraints()[0].String("test-pkg")).To(Equal("test-pkg permits at most 1 of cool-package-2-1-entity, cool-package-2-0-entity"))
				Expect(satVars[2].Identifier().String()).To(Equal("cool-package-3 uniqueness"))
				Expect(satVars[2].Constraints()[0].String("test-pkg")).To(Equal("test-pkg permits at most 1 of cool-package-3-entity"))
			})
			It("forwards any error given by the entity querier", func() {
				mockQuerier.testError = errors.New("oh no!")
				satVars, err := olm.PackageUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("oh no!"))
				Expect(satVars).Should(HaveLen(0))
			})
			It("returns an empty sat.Variable slice when package name key is missing from all entities", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-3-entity", map[string]string{
						"wrong-key":        "cool-package-3",
						propertyOLMVersion: "3.1.2",
						propertyOLMChannel: "channel-2",
					}),
				}
				satVars, err := olm.PackageUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(0))
			})
		})
		Describe("GVKUniqueness", func() {
			var (
				ctx         context.Context
				mockQuerier MockQuerier
			)
			BeforeEach(func() {
				ctx = context.Background()
				mockQuerier = MockQuerier{
					testError:      nil,
					testEntityList: defaultTestEntityList(),
				}
			})
			It("returns a slice of sat.Variable grouped by group, version, and kind, with constraints ordered by package name", func() {
				satVars, err := olm.GVKUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(2))
				sort.Slice(satVars, func(i, j int) bool {
					return satVars[i].Identifier().String() < satVars[j].Identifier().String()
				})
				Expect(satVars[0].Identifier().String()).To(Equal("my-group/my-version/my-kind uniqueness"))
				Expect(satVars[0].Constraints()[0].String("foo")).To(Equal("foo permits at most 1 of cool-package-1-entity, cool-package-2-0-entity, cool-package-3-entity"))
				Expect(satVars[1].Identifier().String()).To(Equal("my-other-group/my-version/my-kind uniqueness"))
				Expect(satVars[1].Constraints()[0].String("foo")).To(Equal("foo permits at most 1 of cool-package-2-1-entity"))
			})
			It("forwards any error given by the entity querier", func() {
				mockQuerier.testError = errors.New("oh no!")
				satVars, err := olm.GVKUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("oh no!"))
				Expect(satVars).Should(HaveLen(0))
			})
			It("returns an empty sat.Variable slice when gvk key is missing from all entities", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-3-entity", map[string]string{
						propertyOLMPackageName: "cool-package-3",
						"wrong-key":            "{\"group\":\"my-group\",\"version\":\"my-version\",\"kind\":\"my-kind\"}",
					}),
				}
				satVars, err := olm.GVKUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(0))
			})
			It("returns an empty sat.Variable slice when gvk field is malformed in all entities", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-3-entity", map[string]string{
						propertyOLMPackageName: "cool-package-3",
						propertyOLMGVK:         "abcdefg",
					}),
				}
				satVars, err := olm.GVKUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(0))
			})
			It("does not panic but returns a constraint with '//' id when gvk json is missing fields", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-3-entity", map[string]string{
						propertyOLMPackageName: "cool-package-3",
						propertyOLMGVK:         "{}",
					}),
				}
				satVars, err := olm.GVKUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Identifier().String()).To(Equal("// uniqueness"))
			})
		})
	})
	Context("dependency", func() {
		Describe("PackageDependency", func() {
			var (
				ctx         context.Context
				mockQuerier MockQuerier
			)
			BeforeEach(func() {
				ctx = context.Background()
				mockQuerier = MockQuerier{
					testError:      nil,
					testEntityList: defaultTestEntityList(),
				}
			})
			It("returns one satVar containing an constraint which lists all available dependencies", func() {
				satVars, err := olm.PackageDependency("cool-package-2-dep", "cool-package-2", "<=3.0.2").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Identifier().String()).To(Equal("cool-package-2-dep"))
				Expect(satVars[0].Constraints()).Should(HaveLen(1))
				msg := satVars[0].Constraints()[0].String("test-pkg")
				Expect(msg).To(Equal("test-pkg requires at least one of cool-package-2-1-entity, cool-package-2-0-entity"))
			})
			It("forwards any error encountered by the entity querier", func() {
				mockQuerier.testError = errors.New("oh no!")
				satVars, err := olm.PackageDependency("cool-package-1-dep", "cool-package-1", ">1.0.0").GetVariables(ctx, mockQuerier)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("oh no!"))
				Expect(satVars).Should(HaveLen(0))
			})
		})
		Describe("GVKDependency", func() {
			var (
				ctx         context.Context
				mockQuerier MockQuerier
			)
			BeforeEach(func() {
				ctx = context.Background()
				mockQuerier = MockQuerier{
					testError:      nil,
					testEntityList: defaultTestEntityList(),
				}
			})
			It("returns a single satVar which lists all available dependencies based on gvk", func() {
				satVars, err := olm.GVKDependency("cool-package-2-dep", "my-group", "my-version", "my-kind").GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				Expect(satVars[0].Identifier().String()).To(Equal("cool-package-2-dep"))
				Expect(satVars[0].Constraints()).Should(HaveLen(1))
				msg := satVars[0].Constraints()[0].String("test-pkg")
				Expect(msg).To(Equal("test-pkg requires at least one of cool-package-1-entity, cool-package-2-0-entity, cool-package-3-entity"))
			})
			It("forwards any error encountered by the entity querier", func() {
				mockQuerier.testError = errors.New("oh no!")
				satVars, err := olm.GVKDependency("cool-package-2-dep", "my-group", "my-version", "my-kind").GetVariables(ctx, mockQuerier)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("oh no!"))
				Expect(satVars).Should(HaveLen(0))
			})
		})
	})
	Context("byChannelAndVersion", func() {
		var (
			ctx         context.Context
			mockQuerier MockQuerier
		)
		BeforeEach(func() {
			ctx = context.Background()
			mockQuerier = MockQuerier{
				testError: nil,
			}
		})
		DescribeTable("package name ordering", func(pkg1NameKey string, pkg2NameKey string, matchElements Elements) {
			mockQuerier.testEntityList = entitysource.EntityList{
				*entitysource.NewEntity("cool-package-entity-1", map[string]string{
					pkg1NameKey:               "cool-package-1",
					propertyOLMChannel:        "channel-1",
					propertyOLMGVK:            "{\"group\":\"my-group\",\"version\":\"my-version\",\"kind\":\"my-kind\"}",
					propertyOLMDefaultChannel: "channel-1",
				}),
				*entitysource.NewEntity("cool-package-entity-2", map[string]string{
					pkg2NameKey:               "cool-package-2",
					propertyOLMChannel:        "channel-1",
					propertyOLMGVK:            "{\"group\":\"my-group\",\"version\":\"my-version\",\"kind\":\"my-kind\"}",
					propertyOLMDefaultChannel: "channel-1",
				}),
			}
			satVars, err := olm.GVKUniqueness().GetVariables(ctx, mockQuerier)
			Expect(err).NotTo(HaveOccurred())
			Expect(satVars).Should(HaveLen(1))
			entities := strings.Split(satVars[0].Constraints()[0].String("pkg"), ", ")
			Expect(entities).To(MatchAllElementsWithIndex(IndexIdentity, matchElements))
		},
			Entry("orders by packageName when both keys exist", propertyOLMPackageName, propertyOLMPackageName, Elements{
				"0": Equal("pkg permits at most 1 of cool-package-entity-1"),
				"1": Equal("cool-package-entity-2"),
			}),
			Entry("orders with entity-2 first when entity-1 is missing packageName", "wrong-key", propertyOLMPackageName, Elements{
				"0": Equal("pkg permits at most 1 of cool-package-entity-2"),
				"1": Equal("cool-package-entity-1"),
			}),
			Entry("orders with entity-1 first when entity-2 is missing packageName", propertyOLMPackageName, "wrong-key", Elements{
				"0": Equal("pkg permits at most 1 of cool-package-entity-1"),
				"1": Equal("cool-package-entity-2"),
			}),
		)
		Describe("channel and version ordering", func() {
			It("orders sat vars with identical packageName by channel and version in that order of priority", func() {
				mockQuerier.testEntityList = entitysource.EntityList{
					*entitysource.NewEntity("cool-package-1-ch1-1.0-entity", map[string]string{
						propertyOLMPackageName:    "cool-package-1",
						propertyOLMVersion:        "1.0.1",
						propertyOLMChannel:        "channel-1",
						propertyOLMDefaultChannel: "channel-1",
					}),
					*entitysource.NewEntity("cool-package-1-ch1-invalid-version-a-entity", map[string]string{
						propertyOLMPackageName:    "cool-package-1",
						propertyOLMVersion:        "abcdefg",
						propertyOLMChannel:        "channel-1",
						propertyOLMDefaultChannel: "channel-1",
					}),
					*entitysource.NewEntity("cool-package-1-ch2-versionless-entity", map[string]string{
						propertyOLMPackageName:    "cool-package-1",
						propertyOLMChannel:        "channel-2",
						propertyOLMDefaultChannel: "channel-1",
					}),
					*entitysource.NewEntity("cool-package-1-ch1-1.1-entity", map[string]string{
						propertyOLMPackageName: "cool-package-1",
						propertyOLMVersion:     "1.1.3",
						propertyOLMChannel:     "channel-1",
					}),
					*entitysource.NewEntity("cool-package-1-ch1-invalid-version-b-entity", map[string]string{
						propertyOLMPackageName: "cool-package-1",
						propertyOLMVersion:     "abcdefg",
						propertyOLMChannel:     "channel-1",
					}),
					*entitysource.NewEntity("cool-package-1-ch2-1.2-entity", map[string]string{
						propertyOLMPackageName:    "cool-package-1",
						propertyOLMVersion:        "1.2.3",
						propertyOLMChannel:        "channel-2",
						propertyOLMDefaultChannel: "channel-1",
					}),
					*entitysource.NewEntity("cool-package-1-ch3-1.2-entity", map[string]string{
						propertyOLMPackageName:    "cool-package-1",
						propertyOLMVersion:        "1.2.3",
						propertyOLMChannel:        "channel-3",
						propertyOLMDefaultChannel: "channel-2",
					}),
					*entitysource.NewEntity("cool-package-1-channelless-1.1-entity", map[string]string{
						propertyOLMPackageName: "cool-package-1",
						propertyOLMVersion:     "1.1.3",
					}),
					*entitysource.NewEntity("cool-package-1-ch1-versionless-entity", map[string]string{
						propertyOLMPackageName:    "cool-package-1",
						propertyOLMChannel:        "channel-1",
						propertyOLMDefaultChannel: "channel-1",
					}),
				}
				satVars, err := olm.PackageUniqueness().GetVariables(ctx, mockQuerier)
				Expect(err).NotTo(HaveOccurred())
				Expect(satVars).Should(HaveLen(1))
				entities := strings.Split(satVars[0].Constraints()[0].String("pkg"), ", ")
				Expect(entities).To(MatchAllElementsWithIndex(IndexIdentity, Elements{
					// channel-1 first, ordered by version, versionless last
					"0": Equal("pkg permits at most 1 of cool-package-1-ch1-1.0-entity"),
					"1": Equal("cool-package-1-ch1-invalid-version-a-entity"),
					"2": Equal("cool-package-1-ch1-1.1-entity"),
					"3": Equal("cool-package-1-ch1-invalid-version-b-entity"),
					"4": Equal("cool-package-1-ch1-versionless-entity"),
					// channel-2 second, ordered by version, versionless last
					"5": Equal("cool-package-1-ch2-1.2-entity"),
					"6": Equal("cool-package-1-ch2-versionless-entity"),
					// channel-3 third
					"7": Equal("cool-package-1-ch3-1.2-entity"),
					// channelless last
					"8": Equal("cool-package-1-channelless-1.1-entity"),
				}))
			})
		})
	})
})
