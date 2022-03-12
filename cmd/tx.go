package main

import (
	"fmt"
	"os"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/spf13/cobra"
)

const flagFrom = "from"
const flagTo = "to"
const flagValue = "value"
const flagData = "data"

func txCmd() *cobra.Command {
	var txCmd = &cobra.Command{
		Use:   "tx",
		Short: "Interact with transactions",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	txCmd.AddCommand(txAddCmd())

	return txCmd
}

func txAddCmd() *cobra.Command {
	var txAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add new transaction",
		Run: func(cmd *cobra.Command, args []string) {
			from, _ := cmd.Flags().GetString(flagFrom)
			to, _ := cmd.Flags().GetString(flagTo)
			value, _ := cmd.Flags().GetUint(flagValue)
			data, _ := cmd.Flags().GetString(flagData)

			tx := database.NewTX(database.NewAccount(from), database.NewAccount(to), value, data)

			state, err := database.NewStateFromDisk()
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}

			if err := state.Add(*tx); err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}

			if _, err := state.Persist(); err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Println("TX successfully added to the ledger.")
		},
	}

	txAddCmd.Flags().String(flagFrom, "", "From account")
	txAddCmd.MarkFlagRequired(flagFrom)

	txAddCmd.Flags().String(flagTo, "", "To account")
	txAddCmd.MarkFlagRequired(flagTo)

	txAddCmd.Flags().Uint(flagValue, 0, "Amount tokens")
	txAddCmd.MarkFlagRequired(flagValue)

	txAddCmd.Flags().String(flagData, "", "Possible values: 'reward'")

	return txAddCmd
}
