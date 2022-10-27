package sat

import (
	"fmt"
	"strings"

	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"

	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
)

type DuplicateIdentifier pkgconstraints.Identifier

func (e DuplicateIdentifier) Error() string {
	return fmt.Sprintf("duplicate identifier %q in input", pkgconstraints.Identifier(e))
}

type inconsistentlitMapping []error

func (inconsistentlitMapping) Error() string {
	return "internal solver failure"
}

// zeroVariable is returned by VariableOf in error cases.
type zeroVariable struct{}

var _ pkgconstraints.IVariable = zeroVariable{}

func (zeroVariable) Identifier() pkgconstraints.Identifier {
	return ""
}

func (zeroVariable) Constraints() []pkgconstraints.Constraint {
	return nil
}

// litMapping performs translation between the input and output types of
// Solve (Constraints, Variables, etc.) and the variables that
// appear in the SAT formula.
type litMapping struct {
	inorder     []pkgconstraints.IVariable
	variables   map[z.Lit]pkgconstraints.IVariable
	lits        map[pkgconstraints.Identifier]z.Lit
	constraints map[z.Lit]pkgconstraints.AppliedConstraint
	c           *logic.C
	errs        inconsistentlitMapping
}

// NewlitMapping returns a new litMapping with its state initialized based on
// the provided slice of Variables. This includes construction of
// the translation tables between Variables/Constraints and the
// inputs to the underlying solver.
func newLitMapping(variables []pkgconstraints.IVariable) (*litMapping, error) {
	d := litMapping{
		inorder:     variables,
		variables:   make(map[z.Lit]pkgconstraints.IVariable, len(variables)),
		lits:        make(map[pkgconstraints.Identifier]z.Lit, len(variables)),
		constraints: make(map[z.Lit]pkgconstraints.AppliedConstraint),
		c:           logic.NewCCap(len(variables)),
	}

	// First pass to assign lits:
	for _, variable := range variables {
		im := d.c.Lit()
		if _, ok := d.lits[variable.Identifier()]; ok {
			return nil, DuplicateIdentifier(variable.Identifier())
		}
		d.lits[variable.Identifier()] = im
		d.variables[im] = variable
	}

	for _, variable := range variables {
		for _, constraint := range variable.Constraints() {
			var m z.Lit
			switch constraint.ConstraintType {
			case "mandatory":
				m = d.LitOf(variable.Identifier())
			case "prohibited":
				m = d.LitOf(variable.Identifier()).Not()
			case "not":
				m = d.LitOf(variable.Identifier()).Not()
			case "dependency":
				m = d.LitOf(variable.Identifier()).Not()
				for _, each := range constraint.Properties["ids"].([]pkgconstraints.Identifier) {
					m = d.c.Or(m, d.LitOf(each))
				}
			case "conflict":
				m = d.c.Or(d.LitOf(variable.Identifier()).Not(), d.LitOf(constraint.Properties["id"].(pkgconstraints.Identifier)).Not())
			case "atmost":
				ms := make([]z.Lit, len(constraint.Properties["ids"].([]pkgconstraints.Identifier)))
				for i, each := range constraint.Properties["ids"].([]pkgconstraints.Identifier) {
					ms[i] = d.LitOf(each)
				}
				m = d.c.CardSort(ms).Leq(constraint.Properties["n"].(int))
			case "or":
				subjectLit := d.LitOf(variable.Identifier())
				_, ok := constraint.Properties["issubjectnegated"]
				if ok && constraint.Properties["issubjectnegated"].(bool) {
					subjectLit = subjectLit.Not()
				}
				var operandLit z.Lit
				_, ok = constraint.Properties["id"]
				if ok {
					operandLit = d.LitOf(constraint.Properties["id"].(pkgconstraints.Identifier))
					_, ok = constraint.Properties["isoperandnegated"]
					if ok && constraint.Properties["isoperandnegated"].(bool) {
						operandLit = operandLit.Not()
					}
				}
				m = d.c.Or(subjectLit, operandLit)
			}
			if m == z.LitNull {
				// This constraint doesn't have a
				// useful representation in the SAT
				// inputs.
				continue
			}

			d.constraints[m] = pkgconstraints.AppliedConstraint{
				Variable:   variable,
				Constraint: constraint,
			}
		}
	}

	return &d, nil
}

// LitOf returns the positive literal corresponding to the Variable
// with the given Identifier.
func (d *litMapping) LitOf(id pkgconstraints.Identifier) z.Lit {
	m, ok := d.lits[id]
	if ok {
		return m
	}
	d.errs = append(d.errs, fmt.Errorf("variable %q referenced but not provided", id))
	return z.LitNull
}

// VariableOf returns the Variable corresponding to the provided
// literal, or a zeroVariable if no such Variable exists.
func (d *litMapping) VariableOf(m z.Lit) pkgconstraints.IVariable {
	i, ok := d.variables[m]
	if ok {
		return i
	}
	d.errs = append(d.errs, fmt.Errorf("no variable corresponding to %s", m))
	return zeroVariable{}
}

// ConstraintOf returns the constraint application corresponding to
// the provided literal, or a zeroConstraint if no such constraint
// exists.
func (d *litMapping) ConstraintOf(m z.Lit) pkgconstraints.AppliedConstraint {
	if a, ok := d.constraints[m]; ok {
		return a
	}
	d.errs = append(d.errs, fmt.Errorf("no constraint corresponding to %s", m))
	return pkgconstraints.AppliedConstraint{
		Variable:   zeroVariable{},
		Constraint: pkgconstraints.Constraint{ConstraintType: "zero", Order: nil},
	}
}

// Error returns a single error value that is an aggregation of all
// errors encountered during a litMapping's lifetime, or nil if there have
// been no errors. A non-nil return value likely indicates a problem
// with the solver or constraint implementations.
func (d *litMapping) Error() error {
	if len(d.errs) == 0 {
		return nil
	}
	s := make([]string, len(d.errs))
	for i, err := range d.errs {
		s[i] = err.Error()
	}
	return fmt.Errorf("%d errors encountered: %s", len(s), strings.Join(s, ", "))
}

// AddConstraints adds the current constraints encoded in the embedded circuit to the
// solver g
func (d *litMapping) AddConstraints(g inter.S) {
	d.c.ToCnf(g)
}

func (d *litMapping) AssumeConstraints(s inter.S) {
	for m := range d.constraints {
		s.Assume(m)
	}
}

// CardinalityConstrainer constructs a sorting network to provide
// cardinality constraints over the provided slice of literals. Any
// new clauses and variables are translated to CNF and taught to the
// given inter.Adder, so this function will panic if it is in a test
// context.
func (d *litMapping) CardinalityConstrainer(g inter.Adder, ms []z.Lit) *logic.CardSort {
	clen := d.c.Len()
	cs := d.c.CardSort(ms)
	marks := make([]int8, clen, d.c.Len())
	for i := range marks {
		marks[i] = 1
	}
	for w := 0; w <= cs.N(); w++ {
		marks, _ = d.c.CnfSince(g, marks, cs.Leq(w))
	}
	return cs
}

// AnchorIdentifiers returns a slice containing the Identifiers of
// every Variable with at least one "anchor" constraint, in the
// order they appear in the input.
func (d *litMapping) AnchorIdentifiers() []pkgconstraints.Identifier {
	var ids []pkgconstraints.Identifier
	for _, variable := range d.inorder {
		for _, constraint := range variable.Constraints() {
			if constraint.Anchor {
				ids = append(ids, variable.Identifier())
				break
			}
		}
	}
	return ids
}

func (d *litMapping) Variables(g inter.S) []pkgconstraints.IVariable {
	var result []pkgconstraints.IVariable
	for _, i := range d.inorder {
		if g.Value(d.LitOf(i.Identifier())) {
			result = append(result, i)
		}
	}
	return result
}

func (d *litMapping) Lits(dst []z.Lit) []z.Lit {
	if cap(dst) < len(d.inorder) {
		dst = make([]z.Lit, 0, len(d.inorder))
	}
	dst = dst[:0]
	for _, i := range d.inorder {
		m := d.LitOf(i.Identifier())
		dst = append(dst, m)
	}
	return dst
}

func (d *litMapping) Conflicts(g inter.Assumable) []pkgconstraints.AppliedConstraint {
	whys := g.Why(nil)
	as := make([]pkgconstraints.AppliedConstraint, 0, len(whys))
	for _, why := range whys {
		if a, ok := d.constraints[why]; ok {
			as = append(as, a)
		}
	}
	return as
}
