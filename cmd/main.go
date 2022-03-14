package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var tbbCm = &cobra.Command{
		Use:   "tbb",
		Short: "The Blockchain Bar CLI",
		Run: func(c *cobra.Command, args []string) {
		},
	}

	tbbCm.AddCommand(versionCmd)
	tbbCm.AddCommand(balancesCmd())
	tbbCm.AddCommand(txCmd())
	tbbCm.AddCommand(migrateCmd())
	tbbCm.AddCommand(runCmd())

	err := tbbCm.Execute()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
