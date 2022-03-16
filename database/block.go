package database

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
)

type Hash [32]byte

func (h Hash) MarshalText() ([]byte, error) {
	// return []byte(base64.StdEncoding.EncodeToString(h[:])), nil
	return []byte(hex.EncodeToString(h[:])), nil
}

func (h *Hash) UnmarshalText(data []byte) error {
	// _, err := base64.StdEncoding.Decode(h[:], data)
	_, err := hex.Decode(h[:], data)
	return err
}

func (h Hash) IsEmpty() bool {
	emptyHash := Hash{}
	return bytes.Equal(h[:], emptyHash[:])
}

func (h Hash) IsBlockHashValid() bool {
	return fmt.Sprintf("%x", h[:3]) == "000000" && fmt.Sprintf("%x", h[3]) != "0"
}

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
