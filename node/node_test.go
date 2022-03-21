package node

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/1412335/the-blockchain-bar/wallet"
)

const andrejAccKeystore = "../data/andrej/keystore/UTC--2022-03-21T04-22-25.727222614Z--f57913db69e172c0ad5018fb0cebf63308b2b8d7"
const andrejAccPwd = "123"

const babayagaAccKeystore = "../data/babayaga/keystore/UTC--2022-03-21T04-22-54.946155728Z--ca22e5f9c5ae099f64991ab356826c4d52554bf8"
const babayagaAccPwd = "456"

func getTestDataDirPath() string {
	return path.Join(os.TempDir(), ".tbb")
}

func copyKeystoreFileIntoTestDataDir(dir string, ksFile string) error {
	keyDir := wallet.GetKeystoreDirPath(dir)

	if err := os.MkdirAll(keyDir, os.ModePerm); err != nil {
		return err
	}

	srcFile, err := os.Open(ksFile)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(filepath.Join(keyDir, filepath.Base(ksFile)))
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}

func TestNode_Run(t *testing.T) {
	datadir := getTestDataDirPath()
	err := os.RemoveAll(datadir)
	if err != nil {
		t.Fatal(err)
	}

	n := New(datadir, "127.0.0.1", 8086, database.NewAccount(wallet.AndrejAccount), PeerNode{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := n.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cancel()
}

func TestNode_Mining(t *testing.T) {
	datadir := getTestDataDirPath()
	err := os.RemoveAll(datadir)
	if err != nil {
		t.Fatal(err)
	}

	if err := copyKeystoreFileIntoTestDataDir(datadir, andrejAccKeystore); err != nil {
		t.Fatal(err)
	}

	peer := NewPeerNode("127.0.0.1", 8087, true, true)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

	andrejAcc := database.NewAccount(wallet.AndrejAccount)
	keystoreDir := wallet.GetKeystoreDirPath(datadir)

	n := New(datadir, "127.0.0.1", 8085, andrejAcc, peer)

	errs := make(chan error, 1)

	go func() {
		time.Sleep(time.Second * miningIntervalSecs / 5)

		signedTx, err := wallet.SignTxWithKeystoreAccount(
			database.NewTX(wallet.AndrejAccount, wallet.AndrejAccount, 100, "reward"),
			andrejAcc,
			andrejAccPwd,
			keystoreDir)

		if err != nil {
			errs <- err
			return
		}

		n.AddPendingTX(signedTx, peer)
	}()

	go func() {
		time.Sleep(time.Second*miningIntervalSecs + 5)
		signedTx, err := wallet.SignTxWithKeystoreAccount(
			database.NewTX(wallet.AndrejAccount, wallet.BabayagaAccount, 30, ""),
			andrejAcc,
			andrejAccPwd,
			keystoreDir)

		if err != nil {
			errs <- err
			return
		}

		n.AddPendingTX(signedTx, peer)
	}()

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for range ticker.C {
			if n.state.LatestBlock().Header.Number == 1 {
				cancel()
				close(errs)
				return
			}
		}
	}()

	go func() {
		if err := n.Run(ctx); err != nil {
			errs <- fmt.Errorf("unexpected error: %v", err)
			return
		}

		// run after node closed
		if n.state.LatestBlock().Header.Number != 1 {
			errs <- fmt.Errorf("expected Height=2, got %v", n.state.LatestBlock().Header.Number)
			return
		}
	}()

	err = <-errs
	if err != nil {
		t.Fatal(err)
	}
}

func TestNode_MiningStopOnNewSyncedBlock(t *testing.T) {
	datadir := getTestDataDirPath()
	err := os.RemoveAll(datadir)
	if err != nil {
		t.Fatal(err)
	}

	if err := copyKeystoreFileIntoTestDataDir(datadir, andrejAccKeystore); err != nil {
		t.Fatal(err)
	}
	if err := copyKeystoreFileIntoTestDataDir(datadir, babayagaAccKeystore); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

	andrejAcc := database.NewAccount(wallet.AndrejAccount)
	babayagaAcc := database.NewAccount(wallet.BabayagaAccount)
	keystoreDir := wallet.GetKeystoreDirPath(getTestDataDirPath())

	tx1 := database.NewTX(wallet.AndrejAccount, wallet.BabayagaAccount, 100, "")
	tx2 := database.NewTX(wallet.BabayagaAccount, wallet.AndrejAccount, 40, "")
	tx2Hash, err := tx2.Hash()
	if err != nil {
		t.Fatal(err)
	}

	signedTx1, err := wallet.SignTxWithKeystoreAccount(tx1, andrejAcc, andrejAccPwd, keystoreDir)
	if err != nil {
		t.Fatal(err)
	}
	signedTx2, err := wallet.SignTxWithKeystoreAccount(tx2, babayagaAcc, babayagaAccPwd, keystoreDir)
	if err != nil {
		t.Fatal(err)
	}

	minedBlock, err := Mine(ctx, NewPendingBlock(database.Hash{}, 0, andrejAcc, []database.SignedTx{signedTx1}))
	if err != nil {
		t.Fatal(err)
	}

	peer := NewPeerNode("127.0.0.1", 8087, true, true)
	n := New(datadir, "127.0.0.1", 8088, babayagaAcc, peer)

	errs := make(chan error, 1)

	go func() {
		err := n.AddPendingTX(signedTx1, peer)
		if err != nil {
			errs <- err
			return
		}

		err = n.AddPendingTX(signedTx2, peer)
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		time.Sleep(time.Second*miningIntervalSecs + 2)
		if !n.isMining {
			errs <- fmt.Errorf("node should be mining")
			return
		}

		if _, err := n.state.AddBlock(minedBlock); err != nil {
			errs <- err
			return
		}

		n.newSyncedBlock <- minedBlock

		time.Sleep(time.Second * 2)
		if n.isMining {
			errs <- fmt.Errorf("node should be stop mining")
			return
		}

		_, tx2InPending := n.pendingTxs[tx2Hash.Hex()]

		if len(n.pendingTxs) != 1 || !tx2InPending {
			errs <- fmt.Errorf("missing tx2 in pending")
			return
		}

		time.Sleep(time.Second*miningIntervalSecs + 2)
		if !n.isMining {
			errs <- fmt.Errorf("node should be mining tx2")
			return
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for range ticker.C {
			if n.state.LatestBlock().Header.Number == 1 {
				cancel()
				close(errs)
				return
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 2)
		oldBalances := n.state.Balances

		<-ctx.Done()

		newBalances := n.state.Balances

		expectedAndrejBalance := oldBalances[andrejAcc] - tx1.Value + tx2.Value + database.BlockReward
		expectedBabayagaBalance := oldBalances[babayagaAcc] + tx1.Value + database.BlockReward

		if newBalances[andrejAcc] != expectedAndrejBalance {
			errs <- fmt.Errorf("andrej's balance expected: %d, got: %d", expectedAndrejBalance, newBalances[andrejAcc])
			return
		}

		if newBalances[babayagaAcc] != expectedBabayagaBalance {
			errs <- fmt.Errorf("babayaga's balance expected: %d, got: %d", expectedBabayagaBalance, newBalances[babayagaAcc])
			return
		}

		t.Logf("Starting Andrej balance: %d", oldBalances[andrejAcc])
		t.Logf("Starting BabaYaga balance: %d", oldBalances[babayagaAcc])
		t.Logf("Ending Andrej balance: %d", newBalances[andrejAcc])
		t.Logf("Ending BabaYaga balance: %d", newBalances[babayagaAcc])
	}()

	go func() {
		if err := n.Run(ctx); err != nil {
			errs <- fmt.Errorf("unexpected error: %v", err)
			return
		}
	}()

	err = <-errs
	if err != nil {
		t.Fatal(err)
	}
}
