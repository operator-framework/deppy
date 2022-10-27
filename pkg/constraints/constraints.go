package constraints

import (
	"fmt"
	"strings"
)

// Mandatory returns a Constraint that will permit only solutions that
// contain a particular Variable.
func Mandatory() Constraint {
	return Constraint{
		ConstraintType: "mandatory",
		Anchor:         true,
		Order:          nil,
	}
}

// Prohibited returns a Constraint that will reject any solution that
// contains a particular Variable. Callers may also decide to omit
// an Variable from input to Solve rather than apply such a
// Constraint.
func Prohibited() Constraint {
	return Constraint{
		ConstraintType: "prohibited",
		Anchor:         false,
		Order:          nil,
	}
}

func Not() Constraint {
	return Constraint{
		ConstraintType: "not",
		Anchor:         false,
		Order:          nil,
	}
}

// Dependency returns a Constraint that will only permit solutions
// containing a given Variable on the condition that at least one
// of the Variables identified by the given Identifiers also
// appears in the solution. Identifiers appearing earlier in the
// argument list have higher preference than those appearing later.
func Dependency(ids ...Identifier) Constraint {
	return Constraint{
		ConstraintType: "dependency",
		Properties:     map[string]interface{}{"ids": ids},
		Anchor:         false,
		Order:          ids,
	}
}

// Conflict returns a Constraint that will permit solutions containing
// either the constrained Variable, the Variable identified by
// the given Identifier, or neither, but not both.
func Conflict(id Identifier) Constraint {
	return Constraint{
		ConstraintType: "conflict",
		Properties:     map[string]interface{}{"id": id},
		Anchor:         false,
		Order:          nil,
	}
}

// AtMost returns a Constraint that forbids solutions that contain
// more than n of the Variables identified by the given
// Identifiers.
func AtMost(n int, ids ...Identifier) Constraint {
	return Constraint{
		ConstraintType: "atmost",
		Properties:     map[string]interface{}{"n": n, "ids": ids},
		Anchor:         false,
		Order:          nil,
	}
}

// Or returns a constraints in the form subject OR identifier
// if isSubjectNegated = true, ~subject OR identifier
// if isOperandNegated = true, subject OR ~identifier
// if both are true: ~subject OR ~identifier
func Or(identifier Identifier, isSubjectNegated bool, isOperandNegated bool) Constraint {
	return Constraint{
		ConstraintType: "or",
		Properties:     map[string]interface{}{"id": identifier, "isnubjectnegated": isSubjectNegated, "isoperandnegated": isOperandNegated},
		Anchor:         false,
		Order:          nil,
	}
}

func ConstraintMessage(subject Identifier, constratint Constraint) string {
	switch constratint.ConstraintType {
	case "mandatory":
		return fmt.Sprintf("%s is mandatory", subject)
	case "prohibited":
		return fmt.Sprintf("%s is prohibited", subject)
	case "dependency":
		ids := constratint.Properties["ids"].([]Identifier)
		if len(ids) == 0 {
			return fmt.Sprintf("%s has a dependency without any candidates to satisfy it", subject)
		}
		s := make([]string, len(ids))
		for i, each := range ids {
			s[i] = string(each)
		}
		return fmt.Sprintf("%s requires at least one of %s", subject, strings.Join(s, ", "))
	case "conflict":
		return fmt.Sprintf("%s conflicts with %s", subject, constratint.Properties["id"].(Identifier))
	case "atmost":
		s := make([]string, len(constratint.Properties["ids"].([]Identifier)))
		for i, each := range constratint.Properties["ids"].([]Identifier) {
			s[i] = string(each)
		}
		return fmt.Sprintf("%s permits at most %d of %s", subject, constratint.Properties["n"].(int), strings.Join(s, ", "))
	case "or":
		return fmt.Sprintf("%s is prohibited", subject)
	}
	return ""
}
