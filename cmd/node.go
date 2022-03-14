package main

import (
	"fmt"
	"os"

	"github.com/1412335/the-blockchain-bar/node"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run node & its HTTP API",
		Run: func(cmd *cobra.Command, args []string) {
			dir, err := cmd.Flags().GetString(flagDataDir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			if err := node.Run(dir); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)
	return runCmd
}
