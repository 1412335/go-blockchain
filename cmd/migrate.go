package main

import (
	"fmt"
	"os"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrate tx.db to blocks.db",
		Run: func(cmd *cobra.Command, args []string) {
			dir, err := cmd.Flags().GetString(flagDataDir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			state, err := database.NewStateFromDisk(dir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer state.Close()

			initBlock := database.NewBlock(state.LatestBlockHash(), uint64(time.Now().Unix()), []database.TX{
				database.NewTX("andrej", "babayaga", 2000, ""),
				database.NewTX("andrej", "andrej", 100, "reward"),
				database.NewTX("babayaga", "andrej", 1, ""),
				database.NewTX("babayaga", "caesar", 1000, ""),
				database.NewTX("babayaga", "andrej", 50, ""),
				database.NewTX("andrej", "andrej", 600, "reward"),
			})

			if err = state.AddBlock(initBlock); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			blockHash, err := state.Persist()
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Printf("Accounts balances at: %x\n", blockHash)
		},
	}

	addDefaultRequiredFlags(migrateCmd)

	return migrateCmd
}
