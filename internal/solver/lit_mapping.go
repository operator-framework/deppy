package solver

import (
	"fmt"
	"strings"

	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"
)

type DuplicateIdentifier Identifier

func (e DuplicateIdentifier) Error() string {
	return fmt.Sprintf("duplicate identifier %q in input", Identifier(e))
}

type inconsistentLitMapping []error

func (inconsistentLitMapping) Error() string {
	return "internal solver failure"
}

// litMapping performs translation between the input and output types of
// Solve (Constraints, Constraints, etc.) and the variables that
// appear in the SAT formula.
type litMapping struct {
	inorder              []Constraint
	constraintsByLiteral map[z.Lit][]Constraint
	lits                 map[Identifier]z.Lit
	constraints          map[z.Lit]Constraint
	c                    *logic.C
	errs                 inconsistentLitMapping
}

// newLitMapping returns a new litMapping with its state initialized based on
// the provided slice of Entities. This includes construction of
// the translation tables between Entities/Constraints and the
// inputs to the underlying solver.
func newLitMapping(constraints []Constraint) (*litMapping, error) {
	d := litMapping{
		inorder:              make([]Constraint, len(constraints)),
		constraintsByLiteral: make(map[z.Lit][]Constraint, 0),
		lits:                 make(map[Identifier]z.Lit, 0),
		constraints:          make(map[z.Lit]Constraint),
		c:                    logic.NewC(),
	}

	for i, constraint := range constraints {
		d.inorder[i] = constraints[i]

		m := constraint.apply(d.c, &d)
		if m == z.LitNull {
			// This constraint doesn't have a
			// useful representation in the SAT
			// inputs.
			continue
		}

		d.constraints[m] = constraint
		litForVar := d.lits[constraint.Subject()]
		d.constraintsByLiteral[litForVar] = append(d.constraintsByLiteral[litForVar], constraint)
	}

	return &d, nil
}

// LitOf returns the positive literal corresponding to the *DeppyEntity
// with the given Identifier.
func (d *litMapping) LitOf(id Identifier) z.Lit {
	_, ok := d.lits[id]
	if !ok {
		d.lits[id] = d.c.Lit()
	}
	return d.lits[id]
}

func (d *litMapping) ConstraintsFor(m z.Lit) []Constraint {
	constrs, ok := d.constraintsByLiteral[m]
	if ok {
		return constrs
	}
	return nil
}

// ConstraintOf returns the constraint application corresponding to
// the provided literal, or a zeroConstraint if no such constraint
// exists.
func (d *litMapping) ConstraintOf(m z.Lit) Constraint {
	if a, ok := d.constraints[m]; ok {
		return a
	}
	d.errs = append(d.errs, fmt.Errorf("no constraint corresponding to %s", m))
	return ZeroConstraint
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
	for m, c := range d.constraints {
		if !c.anchor() {
			s.Assume(m)
		}
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
// every variable with at least one "anchor" constraint, in the
// order they appear in the input.
func (d *litMapping) AnchorIdentifiers() []Identifier {
	var ids []Identifier
	for _, c := range d.inorder {
		if c.anchor() {
			ids = append(ids, c.Subject())
		}
	}

	return ids
}

func (d *litMapping) Lits(dst []z.Lit) []z.Lit {
	if cap(dst) < len(d.lits) {
		dst = make([]z.Lit, 0, len(d.lits))
	}
	dst = dst[:0]
	for _, lit := range d.lits {
		dst = append(dst, lit)
	}
	return dst
}

func (d *litMapping) Conflicts(g inter.Assumable) []Constraint {
	whys := g.Why(nil)
	as := make([]Constraint, 0, len(whys))
	for _, why := range whys {
		if a, ok := d.constraints[why]; ok {
			as = append(as, a)
		}
	}
	return as
}

func (d *litMapping) Selection(g inter.S) []Identifier {
	selection := make([]Identifier, 0)
	for id, lit := range d.lits {
		if g.Value(lit) {
			selection = append(selection, id)
		}
	}
	return selection
}

func (d *litMapping) IdentifierOf(lit z.Lit) Identifier {
	for id, literal := range d.lits {
		if literal == lit {
			return id
		}
	}
	return Identifier("nil")
}
