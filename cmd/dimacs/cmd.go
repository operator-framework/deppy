package dimacs

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/operator-framework/deppy/pkg/constraints"
	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/solver"
)

func NewDimacsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "solve <path>",
		Short: "Solves a sat problem given in dimacs format",
		Long: `Solves a sat problem given in dimacs format. For instance:
c
c this is a comment
c header: p cnf <number of variable> <number of clauses> 
p cnf 2 2
c clauses end in zero, negative means 'not'
c 0 (zero) is not a valid literal
1 2 0
1 -2 0
c cnf: (1 or 2) and (1 and not 2)
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(args[0]); errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("file (%s) not found", args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return solve(args[0])
		},
	}
}

func solve(path string) error {
	// open dimacs file
	dimacsFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening dimacs file (%s): %w", path, err)
	}
	defer dimacsFile.Close()

	dimacs, err := NewDimacs(dimacsFile)
	if err != nil {
		return fmt.Errorf("error parsing dimacs file (%s): %w", path, err)
	}

	// build solver
	so, err := solver.NewDeppySolver(entitysource.NewGroup(NewDimacsEntitySource(dimacs)), constraints.NewConstraintAggregator(NewDimacsConstraintGenerator(dimacs)))
	if err != nil {
		return err
	}

	// get solution
	solution, err := so.Solve(context.Background())
	if err != nil {
		fmt.Printf("no solution found: %s\n", err)
	} else {
		fmt.Println("solution found:")
		for entityID, selected := range solution {
			fmt.Printf("%s = %t\n", entityID, selected)
		}
	}

	return nil
}
