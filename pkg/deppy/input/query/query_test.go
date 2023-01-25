package query_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/deppy/pkg/deppy/input/deppyentity"

	"github.com/operator-framework/deppy/pkg/deppy/input/query"
)

func TestPredicates(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Predicate Suite")
}

var _ = Describe("Predicates", func() {
	Describe("Predicate", func() {
		It("should evaluate", func() {
			var pred query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
				_, ok := entity.Properties["test"]
				return ok, nil
			}
			entityOk := deppyentity.NewEntity("1", map[string]string{
				"test": "ok",
			})
			entityNotOk := deppyentity.NewEntity("1", map[string]string{
				"notest": "notok",
			})
			value, err := pred(entityOk)
			Expect(value).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			value, err = pred(entityNotOk)
			Expect(value).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error", func() {
			var pred query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
				_, ok := entity.Properties["test"]
				if !ok {
					return false, fmt.Errorf("property 'test' not found")
				}
				return ok, nil
			}
			entityOk := deppyentity.NewEntity("1", map[string]string{
				"test": "ok",
			})
			entityNotOk := deppyentity.NewEntity("1", map[string]string{
				"notest": "notok",
			})
			value, err := pred(entityOk)
			Expect(value).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			value, err = pred(entityNotOk)
			Expect(value).To(BeFalse())
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(fmt.Errorf("property 'test' not found")))
		})
	})

	Describe("ConfigurableEntity", func() {
		Describe("And", func() {
			It("should return true if all predicates evaluate to true", func() {
				var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				andPred := query.And(predOne, predTwo).Predicate()
				value, err := andPred(nil)
				Expect(value).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})
			It("should return false if one of predicates evaluates to false", func() {
				var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, nil
				}
				var predThree query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				andPred := query.And(predOne, predTwo, predThree).Predicate()
				value, err := andPred(nil)
				Expect(value).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})
			It("should return error when one of predicates evaluations results in an error", func() {
				var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, fmt.Errorf("this is an error")
				}
				var predThree query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				andPred := query.And(predOne, predTwo, predThree).Predicate()
				value, err := andPred(nil)
				Expect(value).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(fmt.Errorf("this is an error")))
			})
			When("TreatErrorAsFalse options is enabled", func() {
				It("should return false if one of predicates evaluations results in an error", func() {
					var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return true, nil
					}
					var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return false, fmt.Errorf("this is an error")
					}
					var predThree query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return true, nil
					}
					andPred := query.And(predOne, predTwo, predThree).WithOptions(query.TreatErrorAsFalse())
					value, err := andPred(nil)
					Expect(value).To(BeFalse())
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Describe("Or", func() {
			It("should return true if any predicates evaluate to true", func() {
				var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, nil
				}
				var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				orPred := query.Or(predOne, predTwo).Predicate()
				value, err := orPred(nil)
				Expect(value).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})
			It("should return false if all of predicates evaluates to false", func() {
				var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, nil
				}
				var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, nil
				}
				var predThree query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, nil
				}
				orPred := query.Or(predOne, predTwo, predThree).Predicate()
				value, err := orPred(nil)
				Expect(value).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})
			It("should return error if one of predicates evaluations results in an error", func() {
				var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, fmt.Errorf("this is an error")
				}
				var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				var predThree query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				orPred := query.Or(predOne, predTwo, predThree).Predicate()
				value, err := orPred(nil)
				Expect(value).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(fmt.Errorf("this is an error")))
			})
			It("should return return true on the first true evaluation", func() {
				var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return true, nil
				}
				var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, fmt.Errorf("some error")
				}
				var predThree query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
					return false, nil
				}
				orPred := query.Or(predOne, predTwo, predThree).Predicate()
				value, err := orPred(nil)
				Expect(value).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})
			When("TreatErrorAsFalse options is enabled", func() {
				It("should return true if any predicate evaluates to true - treating errors as false", func() {
					var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return false, fmt.Errorf("this is an error")
					}
					var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return false, nil
					}
					var predThree query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return true, nil
					}
					orPred := query.Or(predOne, predTwo, predThree).WithOptions(query.TreatErrorAsFalse())
					value, err := orPred(nil)
					Expect(value).To(BeTrue())
					Expect(err).ToNot(HaveOccurred())
				})

				It("should return false if all predicate evaluates to error - treating errors as false", func() {
					var predOne query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return false, fmt.Errorf("this is an error")
					}
					var predTwo query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return false, fmt.Errorf("this is an error")
					}
					var predThree query.Predicate = func(entity *deppyentity.Entity) (bool, error) {
						return true, fmt.Errorf("this is an error")
					}
					orPred := query.Or(predOne, predTwo, predThree).WithOptions(query.TreatErrorAsFalse())
					value, err := orPred(nil)
					Expect(value).To(BeFalse())
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})
})
