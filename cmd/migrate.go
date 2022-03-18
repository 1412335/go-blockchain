package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/1412335/the-blockchain-bar/node"
	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrate blocks.db",
		Run: func(cmd *cobra.Command, args []string) {
			dir, err := cmd.Flags().GetString(flagDataDir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			miner, err := cmd.Flags().GetString(flagMiner)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			peer := node.NewPeerNode("127.0.0.1", 8080, true, true)
			n := node.New(dir, "127.0.0.1", 8080, database.NewAccount(miner), peer)

			n.AddPendingTX(database.NewTX("andrej", "babayaga", 2000, ""), peer)
			n.AddPendingTX(database.NewTX("andrej", "andrej", 100, "reward"), peer)
			n.AddPendingTX(database.NewTX("babayaga", "andrej", 1, ""), peer)
			n.AddPendingTX(database.NewTX("babayaga", "caesar", 1000, ""), peer)
			n.AddPendingTX(database.NewTX("babayaga", "andrej", 50, ""), peer)

			ctx, nodeStop := context.WithTimeout(context.Background(), 10*time.Minute)

			go func() {
				ticker := time.NewTicker(time.Second * 10)
				for {
					select {
					case <-ticker.C:
						if !n.LatestBlockHash().IsEmpty() {
							nodeStop()
							return
						}
					case <-ctx.Done():
						ticker.Stop()
						nodeStop()
						return
					}
				}
			}()

			if err := n.Run(ctx); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Printf("Accounts balances at: %x\n", n.LatestBlockHash())
		},
	}

	addDefaultRequiredFlags(migrateCmd)

	migrateCmd.Flags().String(flagMiner, "", "")

	return migrateCmd
}
