package dimacs

import (
	"context"
	"strings"

	"github.com/operator-framework/deppy/pkg/input"
	"github.com/operator-framework/deppy/pkg/solver"
)

var _ input.VariableSource = &ConstraintGenerator{}

type ConstraintGenerator struct {
	dimacs *Dimacs
}

func NewDimacsVariableSource(dimacs *Dimacs) *ConstraintGenerator {
	return &ConstraintGenerator{
		dimacs: dimacs,
	}
}

func (d *ConstraintGenerator) GetVariables(ctx context.Context, entitySource input.EntitySource) ([]solver.Variable, error) {
	varMap := make(map[solver.Identifier]*input.SimpleVariable, len(d.dimacs.variables))
	variables := make([]solver.Variable, 0, len(d.dimacs.variables))
	if err := entitySource.Iterate(ctx, func(entity *input.Entity) error {
		variable := input.NewSimpleVariable(entity.Identifier())
		variables = append(variables, variable)
		varMap[entity.Identifier()] = variable
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
			variable := varMap[solver.Identifier(strings.TrimPrefix(first, "-"))]
			if strings.HasPrefix(first, "-") {
				variable.AddConstraint(solver.Not())
			} else {
				// TODO: is this the right constraint here? (given that its an achoring constraint?)
				variable.AddConstraint(solver.Mandatory())
			}
			continue
		}
		for i := 1; i < len(terms); i++ {
			variable := varMap[solver.Identifier(strings.TrimPrefix(first, "-"))]
			second := terms[i]
			negSubject := strings.HasPrefix(first, "-")
			negOperand := strings.HasPrefix(second, "-")

			// TODO: this Or constraint is hacky as hell
			variable.AddConstraint(solver.Or(solver.Identifier(strings.TrimPrefix(second, "-")), negSubject, negOperand))
			first = second
		}
	}

	return variables, nil
}
