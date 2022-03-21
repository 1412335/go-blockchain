package node

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/1412335/the-blockchain-bar/wallet"
)

func getTestDataDirPath() string {
	return path.Join(os.TempDir(), ".tbb")
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

	peer := NewPeerNode("127.0.0.1", 8087, true, true)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	n := New(datadir, "127.0.0.1", 8085, database.NewAccount(wallet.AndrejAccount), peer)

	go func() {
		time.Sleep(time.Second * miningIntervalSecs / 5)
		n.AddPendingTX(database.NewTX(wallet.AndrejAccount, wallet.AndrejAccount, 100, "reward"), peer)
	}()

	go func() {
		time.Sleep(time.Second*miningIntervalSecs + 5)
		n.AddPendingTX(database.NewTX(wallet.AndrejAccount, wallet.BabayagaAccount, 30, ""), peer)
	}()

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for range ticker.C {
			if n.state.LatestBlock().Header.Number == 1 {
				cancel()
				return
			}
		}
	}()

	if err := n.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// run after node closed
	if n.state.LatestBlock().Header.Number != 1 {
		t.Fatalf("expected Height=2, got %v", n.state.LatestBlock().Header.Number)
	}
}

func TestNode_MiningStopOnNewSyncedBlock(t *testing.T) {
	datadir := getTestDataDirPath()
	err := os.RemoveAll(datadir)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	tx1 := database.NewTX(wallet.AndrejAccount, wallet.BabayagaAccount, 40, "")
	tx2 := database.NewTX(wallet.AndrejAccount, wallet.AndrejAccount, 100, "reward")
	tx2Hash, _ := tx2.Hash()

	andrejAcc := database.NewAccount(wallet.AndrejAccount)
	babayagaAcc := database.NewAccount(wallet.BabayagaAccount)

	minedBlock, err := Mine(ctx, NewPendingBlock(database.Hash{}, 0, andrejAcc, []database.TX{tx1}))
	if err != nil {
		t.Fatal(err)
	}

	peer := NewPeerNode("127.0.0.1", 8087, true, true)
	n := New(datadir, "127.0.0.1", 8088, babayagaAcc, peer)

	errs := make(chan error, 1)

	go func() {
		err := n.AddPendingTX(tx1, peer)
		if err != nil {
			errs <- err
			return
		}

		err = n.AddPendingTX(tx2, peer)
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
			errs <- fmt.Errorf("Andrej's balance expected: %d, got: %d", expectedAndrejBalance, newBalances[andrejAcc])
			return
		}

		if newBalances[babayagaAcc] != expectedBabayagaBalance {
			errs <- fmt.Errorf("Babayaga's balance expected: %d, got: %d", expectedBabayagaBalance, newBalances[babayagaAcc])
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
