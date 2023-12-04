package constraint

import (
	"fmt"
	"strings"

	"github.com/go-air/gini/z"

	"github.com/operator-framework/deppy/pkg/deppy"
)

type UserFriendlyConstraintMessageFormatter func(constraint deppy.Constraint, subject deppy.Identifier) string

type UserFriendlyConstraint struct {
	deppy.Constraint
	messageFormatter UserFriendlyConstraintMessageFormatter
}

func (constraint *UserFriendlyConstraint) String(subject deppy.Identifier) string {
	return constraint.messageFormatter(constraint, subject)
}

func NewUserFriendlyConstraint(constraint deppy.Constraint, messageFormatter UserFriendlyConstraintMessageFormatter) *UserFriendlyConstraint {
	return &UserFriendlyConstraint{
		Constraint:       constraint,
		messageFormatter: messageFormatter,
	}
}

type MandatoryConstraint struct{}

func (constraint *MandatoryConstraint) String(subject deppy.Identifier) string {
	return fmt.Sprintf("%s is mandatory", subject)
}

func (constraint *MandatoryConstraint) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	return lm.LitOf(subject)
}

func (constraint *MandatoryConstraint) Order() []deppy.Identifier {
	return nil
}

func (constraint *MandatoryConstraint) Anchor() bool {
	return true
}

// Mandatory returns a Constraint that will permit only solutions that
// contain a particular Variable.
func Mandatory() deppy.Constraint {
	return &MandatoryConstraint{}
}

type ProhibitedConstraint struct{}

func (constraint *ProhibitedConstraint) String(subject deppy.Identifier) string {
	return fmt.Sprintf("%s is ProhibitedConstraint", subject)
}

func (constraint *ProhibitedConstraint) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	return lm.LitOf(subject).Not()
}

func (constraint *ProhibitedConstraint) Order() []deppy.Identifier {
	return nil
}

func (constraint *ProhibitedConstraint) Anchor() bool {
	return false
}

// Prohibited returns a Constraint that will reject any solution that
// contains a particular Variable. Callers may also decide to omit
// an Variable from input to Solve rather than Apply such a
// Constraint.
func Prohibited() deppy.Constraint {
	return &ProhibitedConstraint{}
}

func Not() deppy.Constraint {
	return &ProhibitedConstraint{}
}

type DependencyConstraint struct {
	DependencyIDs []deppy.Identifier
}

func (constraint *DependencyConstraint) String(subject deppy.Identifier) string {
	if len(constraint.DependencyIDs) == 0 {
		return fmt.Sprintf("%s has a DependencyConstraint without any candidates to satisfy it", subject)
	}
	s := make([]string, len(constraint.DependencyIDs))
	for i, each := range constraint.DependencyIDs {
		s[i] = string(each)
	}
	return fmt.Sprintf("%s requires at least one of %s", subject, strings.Join(s, ", "))
}

func (constraint *DependencyConstraint) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	m := lm.LitOf(subject).Not()
	for _, each := range constraint.DependencyIDs {
		m = lm.LogicCircuit().Or(m, lm.LitOf(each))
	}
	return m
}

func (constraint *DependencyConstraint) Order() []deppy.Identifier {
	return constraint.DependencyIDs
}

func (constraint *DependencyConstraint) Anchor() bool {
	return false
}

// Dependency returns a Constraint that will only permit solutions
// containing a given Variable on the condition that at least one
// of the Variables identified by the given Identifiers also
// appears in the solution. Identifiers appearing earlier in the
// argument list have higher preference than those appearing later.
func Dependency(ids ...deppy.Identifier) deppy.Constraint {
	return &DependencyConstraint{
		DependencyIDs: ids,
	}
}

type ConflictConstraint struct {
	ConflictingID deppy.Identifier
}

func (constraint *ConflictConstraint) String(subject deppy.Identifier) string {
	return fmt.Sprintf("%s conflicts with %s", subject, constraint.ConflictingID)
}

func (constraint *ConflictConstraint) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	return lm.LogicCircuit().Or(lm.LitOf(subject).Not(), lm.LitOf(constraint.ConflictingID).Not())
}

func (constraint *ConflictConstraint) Order() []deppy.Identifier {
	return nil
}

func (constraint *ConflictConstraint) Anchor() bool {
	return false
}

// Conflict returns a Constraint that will permit solutions containing
// either the constrained Variable, the Variable identified by
// the given Identifier, OrConstraint neither, but not both.
func Conflict(id deppy.Identifier) deppy.Constraint {
	return &ConflictConstraint{
		ConflictingID: id,
	}
}

type AtMostConstraint struct {
	IDs []deppy.Identifier
	N   int
}

func (constraint *AtMostConstraint) String(subject deppy.Identifier) string {
	s := make([]string, len(constraint.IDs))
	for i, each := range constraint.IDs {
		s[i] = string(each)
	}
	return fmt.Sprintf("%s permits at most %d of %s", subject, constraint.N, strings.Join(s, ", "))
}

func (constraint *AtMostConstraint) Apply(lm deppy.LitMapping, _ deppy.Identifier) z.Lit {
	ms := make([]z.Lit, len(constraint.IDs))
	for i, each := range constraint.IDs {
		ms[i] = lm.LitOf(each)
	}
	return lm.LogicCircuit().CardSort(ms).Leq(constraint.N)
}

func (constraint *AtMostConstraint) Order() []deppy.Identifier {
	return nil
}

func (constraint *AtMostConstraint) Anchor() bool {
	return false
}

// AtMost returns a Constraint that forbids solutions that contain
// more than n of the Variables identified by the given
// Identifiers.
func AtMost(n int, ids ...deppy.Identifier) deppy.Constraint {
	return &AtMostConstraint{
		IDs: ids,
		N:   n,
	}
}

type OrConstraint struct {
	Operand          deppy.Identifier
	IsSubjectNegated bool
	IsOperandNegated bool
}

func (constraint *OrConstraint) String(subject deppy.Identifier) string {
	return fmt.Sprintf("%s is ProhibitedConstraint", subject)
}

func (constraint *OrConstraint) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	subjectLit := lm.LitOf(subject)
	if constraint.IsSubjectNegated {
		subjectLit = subjectLit.Not()
	}
	operandLit := lm.LitOf(constraint.Operand)
	if constraint.IsOperandNegated {
		operandLit = operandLit.Not()
	}
	return lm.LogicCircuit().Or(subjectLit, operandLit)
}

func (constraint *OrConstraint) Order() []deppy.Identifier {
	return nil
}

func (constraint *OrConstraint) Anchor() bool {
	return false
}

// Or returns a constraints in the form subject OR identifier
// if isSubjectNegated = true, ~subject OR identifier
// if isOperandNegated = true, subject OR ~identifier
// if both are true: ~subject OR ~identifier
func Or(identifier deppy.Identifier, isSubjectNegated bool, isOperandNegated bool) deppy.Constraint {
	return &OrConstraint{
		Operand:          identifier,
		IsSubjectNegated: isSubjectNegated,
		IsOperandNegated: isOperandNegated,
	}
}
