package constraint

import (
	"fmt"
	"strings"

	"github.com/go-air/gini/z"

	"github.com/operator-framework/deppy/pkg/deppy"
)

type mandatory struct{}

func (constraint mandatory) String(subject deppy.Identifier) string {
	return fmt.Sprintf("%s is mandatory", subject)
}

func (constraint mandatory) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	return lm.LitOf(subject)
}

func (constraint mandatory) Order() []deppy.Identifier {
	return nil
}

func (constraint mandatory) Anchor() bool {
	return true
}

// Mandatory returns a Constraint that will permit only solutions that
// contain a particular Variable.
func Mandatory() deppy.Constraint {
	return mandatory{}
}

type prohibited struct{}

func (constraint prohibited) String(subject deppy.Identifier) string {
	return fmt.Sprintf("%s is prohibited", subject)
}

func (constraint prohibited) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	return lm.LitOf(subject).Not()
}

func (constraint prohibited) Order() []deppy.Identifier {
	return nil
}

func (constraint prohibited) Anchor() bool {
	return false
}

// Prohibited returns a Constraint that will reject any solution that
// contains a particular Variable. Callers may also decide to omit
// an Variable from input to Solve rather than Apply such a
// Constraint.
func Prohibited() deppy.Constraint {
	return prohibited{}
}

func Not() deppy.Constraint {
	return prohibited{}
}

type dependency []deppy.Identifier

func (constraint dependency) String(subject deppy.Identifier) string {
	if len(constraint) == 0 {
		return fmt.Sprintf("%s has a dependency without any candidates to satisfy it", subject)
	}
	s := make([]string, len(constraint))
	for i, each := range constraint {
		s[i] = string(each)
	}
	return fmt.Sprintf("%s requires at least one of %s", subject, strings.Join(s, ", "))
}

func (constraint dependency) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	m := lm.LitOf(subject).Not()
	for _, each := range constraint {
		m = lm.LogicCircuit().Or(m, lm.LitOf(each))
	}
	return m
}

func (constraint dependency) Order() []deppy.Identifier {
	return constraint
}

func (constraint dependency) Anchor() bool {
	return false
}

// Dependency returns a Constraint that will only permit solutions
// containing a given Variable on the condition that at least one
// of the Variables identified by the given Identifiers also
// appears in the solution. Identifiers appearing earlier in the
// argument list have higher preference than those appearing later.
func Dependency(ids ...deppy.Identifier) deppy.Constraint {
	return dependency(ids)
}

type conflict deppy.Identifier

func (constraint conflict) String(subject deppy.Identifier) string {
	return fmt.Sprintf("%s conflicts with %s", subject, constraint)
}

func (constraint conflict) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	return lm.LogicCircuit().Or(lm.LitOf(subject).Not(), lm.LitOf(deppy.Identifier(constraint)).Not())
}

func (constraint conflict) Order() []deppy.Identifier {
	return nil
}

func (constraint conflict) Anchor() bool {
	return false
}

// Conflict returns a Constraint that will permit solutions containing
// either the constrained Variable, the Variable identified by
// the given Identifier, or neither, but not both.
func Conflict(id deppy.Identifier) deppy.Constraint {
	return conflict(id)
}

type leq struct {
	ids []deppy.Identifier
	n   int
}

func (constraint leq) String(subject deppy.Identifier) string {
	s := make([]string, len(constraint.ids))
	for i, each := range constraint.ids {
		s[i] = string(each)
	}
	return fmt.Sprintf("%s permits at most %d of %s", subject, constraint.n, strings.Join(s, ", "))
}

func (constraint leq) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	ms := make([]z.Lit, len(constraint.ids))
	for i, each := range constraint.ids {
		ms[i] = lm.LitOf(each)
	}
	return lm.LogicCircuit().CardSort(ms).Leq(constraint.n)
}

func (constraint leq) Order() []deppy.Identifier {
	return nil
}

func (constraint leq) Anchor() bool {
	return false
}

// AtMost returns a Constraint that forbids solutions that contain
// more than n of the Variables identified by the given
// Identifiers.
func AtMost(n int, ids ...deppy.Identifier) deppy.Constraint {
	return leq{
		ids: ids,
		n:   n,
	}
}

type or struct {
	operand          deppy.Identifier
	isSubjectNegated bool
	isOperandNegated bool
}

func (constraint or) String(subject deppy.Identifier) string {
	return fmt.Sprintf("%s is prohibited", subject)
}

func (constraint or) Apply(lm deppy.LitMapping, subject deppy.Identifier) z.Lit {
	subjectLit := lm.LitOf(subject)
	if constraint.isSubjectNegated {
		subjectLit = subjectLit.Not()
	}
	operandLit := lm.LitOf(constraint.operand)
	if constraint.isOperandNegated {
		operandLit = operandLit.Not()
	}
	return lm.LogicCircuit().Or(subjectLit, operandLit)
}

func (constraint or) Order() []deppy.Identifier {
	return nil
}

func (constraint or) Anchor() bool {
	return false
}

// Or returns a constraints in the form subject OR identifier
// if isSubjectNegated = true, ~subject OR identifier
// if isOperandNegated = true, subject OR ~identifier
// if both are true: ~subject OR ~identifier
func Or(identifier deppy.Identifier, isSubjectNegated bool, isOperandNegated bool) deppy.Constraint {
	return or{
		operand:          identifier,
		isSubjectNegated: isSubjectNegated,
		isOperandNegated: isOperandNegated,
	}
}
