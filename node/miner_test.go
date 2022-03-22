package node

import (
	"context"
	"crypto/ecdsa"
	"os"
	"testing"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/1412335/the-blockchain-bar/wallet"
	"github.com/ethereum/go-ethereum/crypto"
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

func generateKey() (*ecdsa.PrivateKey, error) {
	privkey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return privkey, nil
}

func createRandomPendingBlock2() (PendingBlock, error) {
	privkey, err := generateKey()
	if err != nil {
		return PendingBlock{}, err
	}

	acc := wallet.PublicKeyToAccount(privkey.PublicKey)
	signedTx, err := wallet.SignTx(database.NewTX(acc.Hex(), wallet.BabayagaAccount, 100, ""), privkey)
	if err != nil {
		return PendingBlock{}, err
	}

	return NewPendingBlock(database.Hash{}, 0, acc, []database.SignedTx{signedTx}), nil
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

	return NewPendingBlock(database.Hash{}, 0, andrejAcc, []database.SignedTx{signedTx1, signedTx2}), nil
}

func TestMine(t *testing.T) {
	pb, err := createRandomPendingBlock2()
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
