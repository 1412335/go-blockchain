package main

import (
	"fmt"
	"os"
	"time"

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
			dir, _ := cmd.Flags().GetString(flagDataDir)
			from, _ := cmd.Flags().GetString(flagFrom)
			to, _ := cmd.Flags().GetString(flagTo)
			value, _ := cmd.Flags().GetUint(flagValue)
			data, _ := cmd.Flags().GetString(flagData)

			tx := database.NewTX(from, to, value, data)

			state, err := database.NewStateFromDisk(dir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			// if err := state.AddTx(tx); err != nil {
			// 	fmt.Fprintln(os.Stderr, err)
			// 	os.Exit(1)
			// }

			// if _, err := state.Persist(); err != nil {
			// 	fmt.Fprintln(os.Stderr, err)
			// 	os.Exit(1)
			// }
			nonce, err := database.RandomNonce()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			block := database.NewBlock(state.LatestBlockHash(), state.NextBlockNumber(), uint64(time.Now().Unix()), nonce, []database.TX{tx})

			hash, err := state.AddBlock(block)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Printf("TX successfully added to the ledger at %x\n", hash)
		},
	}

	txAddCmd.Flags().String(flagFrom, "", "From account")
	txAddCmd.MarkFlagRequired(flagFrom)

	txAddCmd.Flags().String(flagTo, "", "To account")
	txAddCmd.MarkFlagRequired(flagTo)

	txAddCmd.Flags().Uint(flagValue, 0, "Amount tokens")
	txAddCmd.MarkFlagRequired(flagValue)

	txAddCmd.Flags().String(flagData, "", "Possible values: 'reward'")

	addDefaultRequiredFlags(txAddCmd)

	return txAddCmd
}
