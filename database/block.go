package database

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"math"
	"math/big"
)

type Block struct {
	Header BlockHeader `json:"header"`
	TXs    []TX        `json:"payload"`
}

type BlockHeader struct {
	Parent Hash   `json:"parent"`
	Number uint64 `json:"number"`
	Time   uint64 `json:"time"`
	Nonce  uint32 `json:"nonce"`
}

type BlockFS struct {
	BlockHash Hash  `json:"hash"`
	Block     Block `json:"block"`
}

func NewBlock(parentHash Hash, number uint64, time uint64, nonce uint32, txs []TX) Block {
	return Block{
		Header: BlockHeader{
			Parent: parentHash,
			Number: number,
			Time:   time,
			Nonce:  nonce,
		},
		TXs: txs,
	}
}

func (b Block) Hash() (Hash, error) {
	blockJSON, err := json.Marshal(b)
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(blockJSON), nil
}

func RandomNonce() (uint32, error) {
	// rand.Seed(time.Now().UnixNano())
	// return rand.Uint32()
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32))
	if err != nil {
		return 0, err
	}
	return uint32(n.Int64()), nil
}
