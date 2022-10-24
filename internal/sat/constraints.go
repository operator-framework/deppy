package sat

import (
	"fmt"
	"strings"

	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"
)

// Constraint implementations limit the circumstances under which a
// particular Variable can appear in a solution.
type Constraint interface {
	String(subject Identifier) string
	Apply(c *logic.C, lm *LitMapping, subject Identifier) z.Lit
	Order() []Identifier
	Anchor() bool
}

// zeroConstraint is returned by ConstraintOf in error cases.
type zeroConstraint struct{}

var _ Constraint = zeroConstraint{}

func (zeroConstraint) String(subject Identifier) string {
	return ""
}

func (zeroConstraint) Apply(c *logic.C, lm *LitMapping, subject Identifier) z.Lit {
	return z.LitNull
}

func (zeroConstraint) Order() []Identifier {
	return nil
}

func (zeroConstraint) Anchor() bool {
	return false
}

// AppliedConstraint values compose a single Constraint with the
// Variable it applies to.
type AppliedConstraint struct {
	Variable   Variable
	Constraint Constraint
}

// String implements fmt.Stringer and returns a human-readable message
// representing the receiver.
func (a AppliedConstraint) String() string {
	return a.Constraint.String(a.Variable.Identifier())
}

type mandatory struct{}

func (constraint mandatory) String(subject Identifier) string {
	return fmt.Sprintf("%s is mandatory", subject)
}

func (constraint mandatory) Apply(_ *logic.C, lm *LitMapping, subject Identifier) z.Lit {
	return lm.LitOf(subject)
}

func (constraint mandatory) Order() []Identifier {
	return nil
}

func (constraint mandatory) Anchor() bool {
	return true
}

// Mandatory returns a Constraint that will permit only solutions that
// contain a particular Variable.
func Mandatory() Constraint {
	return mandatory{}
}

type prohibited struct{}

func (constraint prohibited) String(subject Identifier) string {
	return fmt.Sprintf("%s is prohibited", subject)
}

func (constraint prohibited) Apply(c *logic.C, lm *LitMapping, subject Identifier) z.Lit {
	return lm.LitOf(subject).Not()
}

func (constraint prohibited) Order() []Identifier {
	return nil
}

func (constraint prohibited) Anchor() bool {
	return false
}

// Prohibited returns a Constraint that will reject any solution that
// contains a particular Variable. Callers may also decide to omit
// an Variable from input to Solve rather than Apply such a
// Constraint.
func Prohibited() Constraint {
	return prohibited{}
}

type dependency []Identifier

func (constraint dependency) String(subject Identifier) string {
	if len(constraint) == 0 {
		return fmt.Sprintf("%s has a dependency without any candidates to satisfy it", subject)
	}
	s := make([]string, len(constraint))
	for i, each := range constraint {
		s[i] = string(each)
	}
	return fmt.Sprintf("%s requires at least one of %s", subject, strings.Join(s, ", "))
}

func (constraint dependency) Apply(c *logic.C, lm *LitMapping, subject Identifier) z.Lit {
	m := lm.LitOf(subject).Not()
	for _, each := range constraint {
		m = c.Or(m, lm.LitOf(each))
	}
	return m
}

func (constraint dependency) Order() []Identifier {
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
func Dependency(ids ...Identifier) Constraint {
	return dependency(ids)
}

type conflict Identifier

func (constraint conflict) String(subject Identifier) string {
	return fmt.Sprintf("%s conflicts with %s", subject, constraint)
}

func (constraint conflict) Apply(c *logic.C, lm *LitMapping, subject Identifier) z.Lit {
	return c.Or(lm.LitOf(subject).Not(), lm.LitOf(Identifier(constraint)).Not())
}

func (constraint conflict) Order() []Identifier {
	return nil
}

func (constraint conflict) Anchor() bool {
	return false
}

// Conflict returns a Constraint that will permit solutions containing
// either the constrained Variable, the Variable identified by
// the given Identifier, or neither, but not both.
func Conflict(id Identifier) Constraint {
	return conflict(id)
}

type leq struct {
	ids []Identifier
	n   int
}

func (constraint leq) String(subject Identifier) string {
	s := make([]string, len(constraint.ids))
	for i, each := range constraint.ids {
		s[i] = string(each)
	}
	return fmt.Sprintf("%s permits at most %d of %s", subject, constraint.n, strings.Join(s, ", "))
}

func (constraint leq) Apply(c *logic.C, lm *LitMapping, subject Identifier) z.Lit {
	ms := make([]z.Lit, len(constraint.ids))
	for i, each := range constraint.ids {
		ms[i] = lm.LitOf(each)
	}
	return c.CardSort(ms).Leq(constraint.n)
}

func (constraint leq) Order() []Identifier {
	return nil
}

func (constraint leq) Anchor() bool {
	return false
}

// AtMost returns a Constraint that forbids solutions that contain
// more than n of the Variables identified by the given
// Identifiers.
func AtMost(n int, ids ...Identifier) Constraint {
	return leq{
		ids: ids,
		n:   n,
	}
}
