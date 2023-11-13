package deppy

import (
	"fmt"
	"strings"

	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"
)

// NotSatisfiable is an error composed of a minimal set of applied
// constraints that is sufficient to make a solution impossible.
type NotSatisfiable []AppliedConstraint

func (e NotSatisfiable) Error() string {
	const msg = "constraints not satisfiable"
	if len(e) == 0 {
		return msg
	}
	s := make([]string, len(e))
	for i, a := range e {
		s[i] = a.String()
	}
	return fmt.Sprintf("%s:\n%s", msg, strings.Join(s, "\n"))
}

// Identifier values uniquely identify particular Variables within
// the input to a single call to Solve.
type Identifier string

func (id Identifier) String() string {
	return string(id)
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

// LitMapping performs translation between the input and output types of
// Solve (Constraints, Variables, etc.) and the variables that
// appear in the SAT formula.
type LitMapping interface {
	LitOf(subject Identifier) z.Lit
	LogicCircuit() *logic.C
}

// Constraint implementations limit the circumstances under which a
// particular Variable can appear in a solution.
type Constraint interface {
	String(subject Identifier) string
	Apply(lm LitMapping, subject Identifier) z.Lit
	Order() []Identifier
	Anchor() bool
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
