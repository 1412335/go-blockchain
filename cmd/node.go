package main

import (
	"context"
	"fmt"
	"os"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/1412335/the-blockchain-bar/node"
	"github.com/spf13/cobra"
)

const flagIP = "ip"
const flagPort = "port"
const flagMiner = "miner"

const DefaultIP = "127.0.0.1"
const DefaultHTTPort = 8080

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run node & its HTTP API",
		Run: func(cmd *cobra.Command, args []string) {
			ip, err := cmd.Flags().GetString(flagIP)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			port, err := cmd.Flags().GetUint64(flagPort)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

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

			bootstrap := node.NewPeerNode("127.0.0.1", 8080, true, false)

			n := node.New(dir, ip, port, database.NewAccount(miner), bootstrap)
			if err := n.Run(context.Background()); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)

	runCmd.Flags().String(flagIP, DefaultIP, "")
	runCmd.Flags().Uint64(flagPort, DefaultHTTPort, "")

	runCmd.Flags().String(flagMiner, "", "")

	return runCmd
}
