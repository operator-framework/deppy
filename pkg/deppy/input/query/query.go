package query

import (
	"github.com/operator-framework/deppy/pkg/deppy/input/deppyentity"
)

// Predicate returns true if the entity should be kept when filtering
type Predicate func(entity *deppyentity.Entity) (bool, error)

// PredicateOption allows the predicate config to be mutated
type PredicateOption func(config *predicateConfig)

// TreatErrorAsFalse is a PredicateOption which instructs predicate
// evaluation to treat errors as 'false'
func TreatErrorAsFalse() PredicateOption {
	return func(config *predicateConfig) {
		config.treatErrorAsFalse = true
	}
}

// predicateConfig defines options for how predicates should be evaluated
type predicateConfig struct {
	treatErrorAsFalse bool
}

func (c *predicateConfig) apply(options []PredicateOption) {
	for _, opt := range options {
		opt(c)
	}
}

// defaultPredicateConfig returns the default predicate config
func defaultPredicateConfig() *predicateConfig {
	return &predicateConfig{
		treatErrorAsFalse: false,
	}
}

// ConfigurablePredicate can be given PredicateOptions to control the way
// predicates get evaluated
type ConfigurablePredicate interface {
	Predicate() Predicate
	WithOptions(options ...PredicateOption) Predicate
}

type and struct {
	predicates []Predicate
	cfg        *predicateConfig
}

func (a *and) Predicate() Predicate {
	return func(entity *deppyentity.Entity) (bool, error) {
		eval := true
		for _, predicate := range a.predicates {
			value, err := predicate(entity)
			if err != nil {
				if !a.cfg.treatErrorAsFalse {
					return false, err
				}
				value = false
			}
			eval = eval && value
			if !eval {
				return false, nil
			}
		}
		return eval, nil
	}
}

func (a *and) WithOptions(options ...PredicateOption) Predicate {
	a.cfg.apply(options)
	return a.Predicate()
}

func And(predicates ...Predicate) ConfigurablePredicate {
	return &and{
		predicates: predicates,
		cfg:        defaultPredicateConfig(),
	}
}

type or struct {
	predicates []Predicate
	cfg        *predicateConfig
}

func (o *or) Predicate() Predicate {
	return func(entity *deppyentity.Entity) (bool, error) {
		eval := false
		for _, predicate := range o.predicates {
			value, err := predicate(entity)
			if err != nil {
				if !o.cfg.treatErrorAsFalse {
					return false, err
				}
				value = false
			}
			eval = eval || value
			if eval {
				return true, nil
			}
		}
		return eval, nil
	}
}

func (o *or) WithOptions(options ...PredicateOption) Predicate {
	o.cfg.apply(options)
	return o.Predicate()
}

func Or(predicates ...Predicate) ConfigurablePredicate {
	return &or{
		predicates: predicates,
		cfg:        defaultPredicateConfig(),
	}
}

func Not(predicate Predicate) Predicate {
	return func(entity *deppyentity.Entity) (bool, error) {
		value, err := predicate(entity)
		if err != nil {
			return false, err
		}
		return !value, nil
	}
}
