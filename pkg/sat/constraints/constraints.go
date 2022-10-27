package constraints

import (
	internalsat "github.com/operator-framework/deppy/internal/sat"
	pkgsat "github.com/operator-framework/deppy/pkg/sat"
)

// Mandatory returns a Constraint that will permit only solutions that
// contain a particular Variable.
func Mandatory() pkgsat.Constraint {
	return internalsat.Mandatory()
}

// Prohibited returns a Constraint that will reject any solution that
// contains a particular Variable. Callers may also decide to omit
// an Variable from input to Solve rather than apply such a
// Constraint.
func Prohibited() pkgsat.Constraint {
	return internalsat.Prohibited()
}

func Not() pkgsat.Constraint {
	return internalsat.Not()
}

// Dependency returns a Constraint that will only permit solutions
// containing a given Variable on the condition that at least one
// of the Variables identified by the given Identifiers also
// appears in the solution. Identifiers appearing earlier in the
// argument list have higher preference than those appearing later.
func Dependency(ids ...pkgsat.Identifier) pkgsat.Constraint {
	return internalsat.Dependency(ids)
}

// Conflict returns a Constraint that will permit solutions containing
// either the constrained Variable, the Variable identified by
// the given Identifier, or neither, but not both.
func Conflict(id pkgsat.Identifier) pkgsat.Constraint {
	return internalsat.Conflict(id)
}

// AtMost returns a Constraint that forbids solutions that contain
// more than n of the Variables identified by the given
// Identifiers.
func AtMost(n int, ids ...pkgsat.Identifier) pkgsat.Constraint {
	return internalsat.AtMost(n, ids)
}

// Or returns a constraints in the form subject OR identifier
// if isSubjectNegated = true, ~subject OR identifier
// if isOperandNegated = true, subject OR ~identifier
// if both are true: ~subject OR ~identifier
func Or(identifier pkgsat.Identifier, isSubjectNegated bool, isOperandNegated bool) pkgsat.Constraint {
	return internalsat.Or(identifier, isSubjectNegated, isOperandNegated)
}
