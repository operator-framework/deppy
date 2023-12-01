package sudoku

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/operator-framework/deppy/pkg/deppy"

	"github.com/operator-framework/deppy/pkg/deppy/solver"
)

func NewSudokuCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sudoku",
		Short: "Returns a solved sudoku board",
		RunE: func(cmd *cobra.Command, args []string) error {
			return solve()
		},
	}
}

func solve() error {
	// build solver
	so, err := solver.New()
	if err != nil {
		return err
	}

	// get solution
	vars, err := GenerateVariables()
	if err != nil {
		return err
	}
	selection, err := so.Solve(vars)
	if err != nil {
		fmt.Println("no solution found")
	} else {
		selected := map[deppy.Identifier]struct{}{}
		for _, variable := range selection {
			selected[variable.Identifier()] = struct{}{}
		}
		for row := 0; row < 9; row++ {
			for col := 0; col < 9; col++ {
				found := false
				for n := 0; n < 9; n++ {
					id := GetID(row, col, n)
					if _, ok := selected[id]; ok {
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
