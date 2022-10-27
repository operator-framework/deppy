package sat

import (
	"fmt"
	"strings"

	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"

	pkgsat "github.com/operator-framework/deppy/pkg/sat"
)

type mandatory struct{}

func (constraint mandatory) String(subject pkgsat.Identifier) string {
	return fmt.Sprintf("%s is mandatory", subject)
}

func (constraint mandatory) Apply(_ *logic.C, lm pkgsat.LitMapping, subject pkgsat.Identifier) z.Lit {
	return lm.LitOf(subject)
}

func (constraint mandatory) Order() []pkgsat.Identifier {
	return nil
}

func (constraint mandatory) Anchor() bool {
	return true
}

// Mandatory returns a Constraint that will permit only solutions that
// contain a particular Variable.
func Mandatory() pkgsat.Constraint {
	return mandatory{}
}

type prohibited struct{}

func (constraint prohibited) String(subject pkgsat.Identifier) string {
	return fmt.Sprintf("%s is prohibited", subject)
}

func (constraint prohibited) Apply(c *logic.C, lm pkgsat.LitMapping, subject pkgsat.Identifier) z.Lit {
	return lm.LitOf(subject).Not()
}

func (constraint prohibited) Order() []pkgsat.Identifier {
	return nil
}

func (constraint prohibited) Anchor() bool {
	return false
}

// Prohibited returns a Constraint that will reject any solution that
// contains a particular Variable. Callers may also decide to omit
// an Variable from input to Solve rather than apply such a
// Constraint.
func Prohibited() pkgsat.Constraint {
	return prohibited{}
}

func Not() pkgsat.Constraint {
	return prohibited{}
}

type dependency []pkgsat.Identifier

func (constraint dependency) String(subject pkgsat.Identifier) string {
	if len(constraint) == 0 {
		return fmt.Sprintf("%s has a dependency without any candidates to satisfy it", subject)
	}
	s := make([]string, len(constraint))
	for i, each := range constraint {
		s[i] = string(each)
	}
	return fmt.Sprintf("%s requires at least one of %s", subject, strings.Join(s, ", "))
}

func (constraint dependency) Apply(c *logic.C, lm pkgsat.LitMapping, subject pkgsat.Identifier) z.Lit {
	m := lm.LitOf(subject).Not()
	for _, each := range constraint {
		m = c.Or(m, lm.LitOf(each))
	}
	return m
}

func (constraint dependency) Order() []pkgsat.Identifier {
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
func Dependency(ids []pkgsat.Identifier) pkgsat.Constraint {
	return dependency(ids)
}

type conflict pkgsat.Identifier

func (constraint conflict) String(subject pkgsat.Identifier) string {
	return fmt.Sprintf("%s conflicts with %s", subject, constraint)
}

func (constraint conflict) Apply(c *logic.C, lm pkgsat.LitMapping, subject pkgsat.Identifier) z.Lit {
	return c.Or(lm.LitOf(subject).Not(), lm.LitOf(pkgsat.Identifier(constraint)).Not())
}

func (constraint conflict) Order() []pkgsat.Identifier {
	return nil
}

func (constraint conflict) Anchor() bool {
	return false
}

// Conflict returns a Constraint that will permit solutions containing
// either the constrained Variable, the Variable identified by
// the given Identifier, or neither, but not both.
func Conflict(id pkgsat.Identifier) pkgsat.Constraint {
	return conflict(id)
}

type leq struct {
	ids []pkgsat.Identifier
	n   int
}

func (constraint leq) String(subject pkgsat.Identifier) string {
	s := make([]string, len(constraint.ids))
	for i, each := range constraint.ids {
		s[i] = string(each)
	}
	return fmt.Sprintf("%s permits at most %d of %s", subject, constraint.n, strings.Join(s, ", "))
}

func (constraint leq) Apply(c *logic.C, lm pkgsat.LitMapping, subject pkgsat.Identifier) z.Lit {
	ms := make([]z.Lit, len(constraint.ids))
	for i, each := range constraint.ids {
		ms[i] = lm.LitOf(each)
	}
	return c.CardSort(ms).Leq(constraint.n)
}

func (constraint leq) Order() []pkgsat.Identifier {
	return nil
}

func (constraint leq) Anchor() bool {
	return false
}

// AtMost returns a Constraint that forbids solutions that contain
// more than n of the Variables identified by the given
// Identifiers.
func AtMost(n int, ids []pkgsat.Identifier) pkgsat.Constraint {
	return leq{
		ids: ids,
		n:   n,
	}
}

type or struct {
	operand          pkgsat.Identifier
	isSubjectNegated bool
	isOperandNegated bool
}

func (constraint or) String(subject pkgsat.Identifier) string {
	return fmt.Sprintf("%s is prohibited", subject)
}

func (constraint or) Apply(c *logic.C, lm pkgsat.LitMapping, subject pkgsat.Identifier) z.Lit {
	subjectLit := lm.LitOf(subject)
	if constraint.isSubjectNegated {
		subjectLit = subjectLit.Not()
	}
	operandLit := lm.LitOf(constraint.operand)
	if constraint.isOperandNegated {
		operandLit = operandLit.Not()
	}
	return c.Or(subjectLit, operandLit)
}

func (constraint or) Order() []pkgsat.Identifier {
	return nil
}

func (constraint or) Anchor() bool {
	return false
}

// Or returns a constraints in the form subject OR identifier
// if isSubjectNegated = true, ~subject OR identifier
// if isOperandNegated = true, subject OR ~identifier
// if both are true: ~subject OR ~identifier
func Or(identifier pkgsat.Identifier, isSubjectNegated bool, isOperandNegated bool) pkgsat.Constraint {
	return or{
		operand:          identifier,
		isSubjectNegated: isSubjectNegated,
		isOperandNegated: isOperandNegated,
	}
}
