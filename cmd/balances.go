package main

import (
	"fmt"
	"os"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/spf13/cobra"
)

func balancesCmd() *cobra.Command {
	var balancesCmd = &cobra.Command{
		Use:   "balances",
		Short: "Interact with balances",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	balancesCmd.AddCommand(balancesListCmd)

	return balancesCmd
}

var balancesListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all balances",
	Run: func(cmd *cobra.Command, args []string) {
		state, err := database.NewStateFromDisk()
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		defer state.Close()

		fmt.Printf("Accounts balances at %x:\n", state.LatestBlockHash())
		fmt.Println("__________________")
		fmt.Println("")
		for account, balance := range state.Balances {
			fmt.Printf("%s: %d\n", account, balance)
		}
	},
}
