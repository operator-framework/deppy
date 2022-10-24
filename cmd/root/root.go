package root

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/deppy/cmd/sudoku"

	"github.com/operator-framework/deppy/cmd/dimacs"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "deppy",
		Short: "Deppy is an open-source constraint solver framework",
		Long: `An open-source constraint solver framework written in Go.
For more information visit https://github.com/operator-framework/deppy`,
	}

	// add sub-commands
	rootCmd.AddCommand(dimacs.NewDimacsCommand())
	rootCmd.AddCommand(sudoku.NewSudokuCommand())

	return rootCmd
}
