package sudoku

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/solver"
)

var Cmd = &cobra.Command{
	Use:   "sudoku",
	Short: "Returns a solved sudoku board",
	RunE: func(cmd *cobra.Command, args []string) error {
		return solve()
	},
}

func solve() error {
	// build solver
	sudoku := NewSudoku()
	so, err := solver.NewDeppySolver(solver.Options{
		EntitySources:     []entitysource.EntitySource{sudoku},
		ConstraintBuilder: constraints.NewDeppyConstraintBuilder(sudoku),
	})

	if err != nil {
		return err
	}

	// get solution
	solution, err := so.Solve(context.Background())
	if err != nil {
		fmt.Println("no solution found")
	} else {
		for row := 0; row < 9; row++ {
			for col := 0; col < 9; col++ {
				found := false
				for n := 0; n < 9; n++ {
					id := GetID(row, col, n)
					if solution[id] {
						fmt.Printf("%d", n+1)
						found = true
						break
					}
				}
				if !found {
					fmt.Printf(" ")
				}
				if col != 8 {
					fmt.Printf(" ")
				}
			}
			fmt.Printf("\n")
		}
	}

	return nil
}
