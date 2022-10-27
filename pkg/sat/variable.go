package sat

import (
	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"
)

// Identifier values uniquely identify particular Variables within
// the input to a single call to Solve.
type Identifier string

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

func (id Identifier) String() string {
	return string(id)
}

// Constraint implementations limit the circumstances under which a
// particular Variable can appear in a solution.
type Constraint interface {
	String(subject Identifier) string
	Apply(c *logic.C, lm LitMapping, subject Identifier) z.Lit
	Order() []Identifier
	Anchor() bool
}

// IdentifierFromString returns an Identifier based on a provided
// string.
func IdentifierFromString(s string) Identifier {
	return Identifier(s)
}

// Variable values are the basic unit of problems and solutions
// understood by this package.
type Variable interface {
	// Identifier returns the Identifier that uniquely identifies
	// this Variable among all other Variables in a given
	// problem.
	Identifier() Identifier
	// Constraints returns the set of constraints that apply to
	// this Variable.
	Constraints() []Constraint
}

// zeroVariable is returned by VariableOf in error cases.
type zeroVariable struct{}

var _ Variable = zeroVariable{}

func (zeroVariable) Identifier() Identifier {
	return ""
}

func (zeroVariable) Constraints() []Constraint {
	return nil
}

// zeroConstraint is returned by ConstraintOf in error cases.
type zeroConstraint struct{}

var _ Constraint = zeroConstraint{}

func (zeroConstraint) String(subject Identifier) string {
	return ""
}

func (zeroConstraint) Apply(c *logic.C, lm LitMapping, subject Identifier) z.Lit {
	return z.LitNull
}

func (zeroConstraint) Order() []Identifier {
	return nil
}

func (zeroConstraint) Anchor() bool {
	return false
}
