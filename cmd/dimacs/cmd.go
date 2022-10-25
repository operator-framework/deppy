package dimacs

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/solver"
)

var Cmd = &cobra.Command{
	Use:   "solve <path>",
	Short: "Solves a sat problem given in dimacs format",
	Long:  `Solves a sat problem given in dimacs format"`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}
		if _, err := os.Stat(args[0]); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("file (%s) not found", args[0])
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return solve(args[0])
	},
}

func solve(path string) error {
	// open dimacs file
	dimacsFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening dimacs file (%s): %w", path, err)
	}

	dimacs, err := NewDimacs(dimacsFile)
	if err != nil {
		return fmt.Errorf("error parsing dimacs file (%s): %w", path, err)
	}

	// build solver
	so, err := solver.NewDeppySolver(solver.Options{
		EntitySources:     []entitysource.EntitySource{NewDimacsEntitySource(dimacs)},
		ConstraintBuilder: constraints.NewDeppyConstraintBuilder(NewDimacsConstraintGenerator(dimacs)),
	})

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
