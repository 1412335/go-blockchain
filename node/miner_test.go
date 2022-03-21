package node

import (
	"context"
	"testing"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/1412335/the-blockchain-bar/wallet"
)

func TestValidBlockHash(t *testing.T) {
	hexHash := "00000028d46b7c1e8d5b5b696c5acd80cac95e6014bc9eec62f2a0a6625501"
	var hash = database.Hash{}

	if err := hash.UnmarshalText([]byte(hexHash)); err != nil {
		t.Fatalf("unable to unmarshal hex hash: %v", err)
	}

	if isValid := hash.IsBlockHashValid(); !isValid {
		t.Fatalf("hash '%s' should be valid", hexHash)
	}
}

func TestInvalidBlockHash(t *testing.T) {
	hexHash := "005d28d46b7c1e8d5b5b696c5acd80cac95e6014bc9eec62f2a0a6625501"
	var hash = database.Hash{}

	if err := hash.UnmarshalText([]byte(hexHash)); err != nil {
		t.Fatalf("unable to unmarshal hex hash: %v", err)
	}

	if isValid := hash.IsBlockHashValid(); isValid {
		t.Fatalf("hash '%s' should not be valid", hexHash)
	}
}

func createRandomPendingBlock() PendingBlock {
	return PendingBlock{
		parent: database.Hash{},
		number: 0,
		time:   uint64(time.Now().Unix()),
		miner:  database.NewAccount(wallet.AndrejAccount),
		txs: []database.TX{
			database.NewTX(wallet.AndrejAccount, wallet.AndrejAccount, 1, ""),
			database.NewTX(wallet.AndrejAccount, wallet.AndrejAccount, 100, "reward"),
		},
	}
}

func TestMine(t *testing.T) {
	pb := createRandomPendingBlock()
	ctx := context.Background()

	minedBlock, err := Mine(ctx, pb)
	if err != nil {
		t.Fatal(err)
	}

	minedBlockHash, err := minedBlock.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !minedBlockHash.IsBlockHashValid() {
		t.Fatal("mined block hash is not valid")
	}
}

func TestMineWithTimeout(t *testing.T) {
	pb := createRandomPendingBlock()
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Millisecond)
	_, err := Mine(ctx, pb)
	if err == nil {
		t.Fatal()
	}
}
