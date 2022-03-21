package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/1412335/the-blockchain-bar/wallet"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/spf13/cobra"
)

const flagKeystoreFile = "ksfile"

func walletCmd() *cobra.Command {
	var walletCmd = &cobra.Command{
		Use:   "wallet",
		Short: "Manages accounts & keys",
		Run: func(c *cobra.Command, args []string) {

		},
	}

	walletCmd.AddCommand(walletNewAccountCmd())
	walletCmd.AddCommand(walletPrintPrivateKeyCmd())

	return walletCmd
}

func walletNewAccountCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "new-account",
		Short: "Create a new account with Private + Public Key",
		Run: func(c *cobra.Command, args []string) {
			password := utils.GetPassPhrase("Please enter password to encrypt new wallet", true)

			dir := getDataDirFromCmd(c)

			acc, err := wallet.NewKeyStoreAccount(dir, password)
			if err != nil {
				fmt.Printf("Error creating account: %v", err)
				os.Exit(1)
			}

			fmt.Printf("New account created: %s", acc.Hex())
		},
	}

	addDefaultRequiredFlags(cmd)

	return cmd
}

func walletPrintPrivateKeyCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "pk-dump",
		Short: "Unlock keystore file & Print Public + Private Key",
		Run: func(c *cobra.Command, args []string) {
			ksFile, _ := c.Flags().GetString(flagKeystoreFile)
			password := utils.GetPassPhrase("Enter password to decrypt the keystore file:", false)

			ksJSON, err := ioutil.ReadFile(ksFile)
			if err != nil {
				fmt.Printf("Read the keystore file failed %v", err)
				os.Exit(1)
			}

			key, err := keystore.DecryptKey(ksJSON, password)
			if err != nil {
				fmt.Printf("Decrypt the keystore file failed %v", err)
				os.Exit(1)
			}

			spew.Dump(key)
		},
	}

	cmd.Flags().String(flagKeystoreFile, "", "keystore file")

	return cmd
}
