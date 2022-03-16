package node

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
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

	n := New(datadir, "127.0.0.1", 8088, PeerNode{})

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

	n := New(datadir, "127.0.0.1", 8088, peer)

	go func() {
		time.Sleep(time.Second * miningIntervalSecs / 5)
		n.AddPendingTX(database.NewTX("andrej", "andrej", 100, "reward"), peer)
	}()

	go func() {
		time.Sleep(time.Second*miningIntervalSecs + 5)
		n.AddPendingTX(database.NewTX("andrej", "babayaga", 30, ""), peer)
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
