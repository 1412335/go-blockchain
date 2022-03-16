package node

import (
	"context"
	"fmt"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
)

type PendingBlock struct {
	parent database.Hash
	number uint64
	time   uint64
	txs    []database.TX
}

func Mine(ctx context.Context, pendingBlock PendingBlock) (database.Block, error) {
	if len(pendingBlock.txs) == 0 {
		return database.Block{}, fmt.Errorf("empty block")
	}

	start := time.Now()
	attempts := 0
	hash := database.Hash{}
	block := database.Block{}

	for !hash.IsBlockHashValid() {
		select {
		case <-ctx.Done():
			return database.Block{}, fmt.Errorf("stop mining after %d attempts with error: %s", attempts, ctx.Err())
		default:
		}

		attempts++

		if attempts%1e6 == 0 || attempts == 1 {
			fmt.Printf("Mining %d pending txs. Attempts #%d\n", len(pendingBlock.txs), attempts)
		}

		nonce, err := database.RandomNonce()
		if err != nil {
			return database.Block{}, err
		}

		block = database.NewBlock(
			pendingBlock.parent,
			pendingBlock.number,
			pendingBlock.time,
			nonce,
			pendingBlock.txs,
		)

		hash, err = block.Hash()
		if err != nil {
			return database.Block{}, fmt.Errorf("can't mine block: %s", err.Error())
		}
	}

	fmt.Printf("Mined new Block using PoW '%x':\n", hash)
	fmt.Printf("\tHeight: %d\n", block.Header.Number)
	fmt.Printf("\tNonce: %d\n", block.Header.Nonce)
	fmt.Printf("\tCreated: %v\n", block.Header.Time)
	// fmt.Printf("\tMiner: %d\n", block.Header.Time)
	fmt.Printf("\tParent: %x\n", block.Header.Parent)
	fmt.Printf("\tAttempts: %d\n", attempts)
	fmt.Printf("\tTime mining: %s\n", time.Since(start))

	return block, nil
}
