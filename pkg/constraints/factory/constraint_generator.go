package cfactory

import (
	internalconstraints "github.com/operator-framework/deppy/internal/constraints"
	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
)

func NewConstraintAggregator(constraintGenerators ...pkgconstraints.ConstraintGenerator) pkgconstraints.ConstraintGenerator {
	return internalconstraints.NewConstraintAggregator(constraintGenerators)
}
