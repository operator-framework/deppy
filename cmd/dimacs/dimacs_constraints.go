package dimacs

import (
	"context"
	"strings"

	"github.com/operator-framework/deppy/pkg/constraints"
	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/sat"
	satconstraints "github.com/operator-framework/deppy/pkg/sat/constraints"
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
	varMap := make(map[entitysource.EntityID]*constraints.Variable, len(d.dimacs.variables))
	variables := make([]sat.Variable, 0, len(d.dimacs.variables))
	if err := querier.Iterate(ctx, func(entity *entitysource.Entity) error {
		variable := constraints.NewVariable(sat.Identifier(entity.ID()))
		variables = append(variables, variable)
		varMap[entity.ID()] = variable
		return nil
	}); err != nil {
		return nil, err
	}

	// create constraints out of the clauses
	for _, clause := range d.dimacs.clauses {
		terms := strings.Split(clause, " ")
		if len(terms) == 0 {
			continue
		}
		first := terms[0]

		if len(terms) == 1 {
			// TODO: check constraints haven't already been added to the variable
			variable := varMap[entitysource.EntityID(strings.TrimPrefix(first, "-"))]
			if strings.HasPrefix(first, "-") {
				variable.AddConstraint(satconstraints.Not())
			} else {
				// TODO: is this the right constraint here? (given that its an achoring constraint?)
				variable.AddConstraint(satconstraints.Mandatory())
			}
			continue
		}
		for i := 1; i < len(terms); i++ {
			variable := varMap[entitysource.EntityID(strings.TrimPrefix(first, "-"))]
			second := terms[i]
			negSubject := strings.HasPrefix(first, "-")
			negOperand := strings.HasPrefix(second, "-")

			// TODO: this Or constraint is hacky as hell
			variable.AddConstraint(satconstraints.Or(sat.Identifier(strings.TrimPrefix(second, "-")), negSubject, negOperand))
			first = second
		}
	}

	return variables, nil
}
