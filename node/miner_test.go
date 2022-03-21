package node

import (
	"context"
	"os"
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

func createRandomPendingBlock() (PendingBlock, error) {
	datadir := getTestDataDirPath()
	err := os.RemoveAll(datadir)
	if err != nil {
		return PendingBlock{}, err
	}

	if err := copyKeystoreFileIntoTestDataDir(datadir, andrejAccKeystore); err != nil {
		return PendingBlock{}, err
	}
	if err := copyKeystoreFileIntoTestDataDir(datadir, babayagaAccKeystore); err != nil {
		return PendingBlock{}, err
	}

	andrejAcc := database.NewAccount(wallet.AndrejAccount)
	babayagaAcc := database.NewAccount(wallet.BabayagaAccount)

	signedTx1, err := wallet.SignTxWithKeystoreAccount(database.NewTX(wallet.AndrejAccount, wallet.BabayagaAccount, 100, ""), andrejAcc, andrejAccPwd, wallet.GetKeystoreDirPath(datadir))
	if err != nil {
		return PendingBlock{}, err
	}

	signedTx2, err := wallet.SignTxWithKeystoreAccount(database.NewTX(wallet.BabayagaAccount, wallet.AndrejAccount, 20, ""), babayagaAcc, babayagaAccPwd, wallet.GetKeystoreDirPath(datadir))
	if err != nil {
		return PendingBlock{}, err
	}

	return PendingBlock{
		parent: database.Hash{},
		number: 0,
		time:   uint64(time.Now().Unix()),
		miner:  database.NewAccount(wallet.AndrejAccount),
		txs: []database.SignedTx{
			signedTx1,
			signedTx2,
		},
	}, nil
}

func TestMine(t *testing.T) {
	pb, err := createRandomPendingBlock()
	if err != nil {
		t.Fatal(err)
	}

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
	pb, err := createRandomPendingBlock()
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Millisecond)
	_, err = Mine(ctx, pb)
	if err == nil {
		t.Fatal()
	}
}
