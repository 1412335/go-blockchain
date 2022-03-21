package main

import (
	"fmt"
	"os"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/spf13/cobra"
)

const flagDataDir = "db"

func balancesCmd() *cobra.Command {
	var balancesCmd = &cobra.Command{
		Use:   "balances",
		Short: "Interact with balances",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	balancesCmd.AddCommand(balancesListCmd())

	return balancesCmd
}

func balancesListCmd() *cobra.Command {
	var balancesListCmd = &cobra.Command{
		Use:   "list",
		Short: "Show all balances",
		Run: func(cmd *cobra.Command, args []string) {
			dir := getDataDirFromCmd(cmd)
			state, err := database.NewStateFromDisk(dir)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}
			defer state.Close()

			fmt.Printf("Accounts balances at %x:\n", state.LatestBlockHash())
			fmt.Println("__________________")
			fmt.Println("")
			for account, balance := range state.Balances {
				fmt.Printf("%s: %d\n", account.Hex(), balance)
			}
		},
	}

	addDefaultRequiredFlags(balancesListCmd)
	return balancesListCmd
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to the database")
	cmd.MarkFlagRequired(flagDataDir)
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dir, err := cmd.Flags().GetString(flagDataDir)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	return dir
}
