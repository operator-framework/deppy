package root

import (
	"fmt"
	"os"

	"github.com/operator-framework/deppy/cmd/dimacs"
	"github.com/operator-framework/deppy/cmd/sudoku"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(dimacs.Cmd)
	rootCmd.AddCommand(sudoku.Cmd)
}

var rootCmd = &cobra.Command{
	Use:   "deppy",
	Short: "Deppy is an open-source constraint solver framework",
	Long: `An open-source constraint solver framework written in Go.
For more information visit https://github.com/operator-framework/deppy`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
