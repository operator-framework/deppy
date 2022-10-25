package dimacs

import (
	"context"
	"strings"

	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"

	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/sat"
)

var _ constraints.ConstraintGenerator = &ConstraintGenerator{}

type ConstraintGenerator struct {
	dimacs *Dimacs
}

func NewDimacsConstraintGenerator(dimacs *Dimacs) *ConstraintGenerator {
	return &ConstraintGenerator{
		dimacs: dimacs,
	}
}

func (d *ConstraintGenerator) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	variables := make([]sat.Variable, 0, len(d.dimacs.variables))
	clauses := make([]sat.Constraint, 0, len(d.dimacs.Clauses()))

	// create solver variables for each variable
	clauses = append(clauses, sat.Mandatory())
	if err := querier.Iterate(ctx, func(entity *entitysource.Entity) error {
		variables = append(variables, constraints.NewVariable(sat.Identifier(entity.ID())))
		return nil
	}); err != nil {
		return nil, err
	}

	// create constraints out of the clauses
	for _, clause := range d.dimacs.clauses {
		clauses = append(clauses, clauseConstraint(strings.Split(clause, " ")))
	}

	variables = append(variables, constraints.NewVariable("cnf", clauses...))
	return variables, nil
}

var _ sat.Constraint = &clauseConstraint{}

// clauseConstraint converts a DIMACS clause (e.g. 1 2 -3, which mean 1 or 2 or not 3)
// into a solver constraint
type clauseConstraint []string

func (constraint clauseConstraint) String(_ sat.Identifier) string {
	return strings.Join(constraint, " or ")
}

func (constraint clauseConstraint) Apply(c *logic.C, lm *sat.LitMapping, _ sat.Identifier) z.Lit {
	lits := make([]z.Lit, len(constraint))
	for i := range constraint {
		if strings.HasPrefix(constraint[i], "-") {
			lits[i] = lm.LitOf(sat.Identifier(constraint[i][1:])).Not()
		} else {
			lits[i] = lm.LitOf(sat.Identifier(constraint[i]))
		}
	}
	return c.Ors(lits...)
}

func (constraint clauseConstraint) Order() []sat.Identifier {
	return nil
}

func (constraint clauseConstraint) Anchor() bool {
	return false
}
