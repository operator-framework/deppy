package root

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deppy",
		Short: "Deppy is an open-source constraint solver framework",
		Long: `An open-source constraint solver framework written in Go.
For more information visit https://github.com/operator-framework/deppy`,
	}
}
