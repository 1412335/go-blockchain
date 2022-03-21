package database

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

type TX struct {
	From  Account `json:"from"`
	To    Account `json:"to"`
	Value uint    `json:"value"`
	Data  string  `json:"data"`
	Time  uint64  `json:"time"`
}

type SignedTx struct {
	TX
	Sign []byte `json:"signature"`
}

func NewTX(from string, to string, value uint, data string) TX {
	return TX{NewAccount(from), NewAccount(to), value, data, uint64(time.Now().Unix())}
}

func (tx *TX) IsReward() bool {
	return tx.Data == "reward"
}

func (tx *TX) Hash() (Hash, error) {
	txJSON, err := tx.Encode()
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(txJSON), nil
}

func (tx *TX) Encode() ([]byte, error) {
	return json.Marshal(tx)
}

func (t *SignedTx) IsAuthentic() (bool, error) {
	txEncoded, err := t.TX.Encode()
	if err != nil {
		return false, err
	}

	pubkey, err := crypto.SigToPub(txEncoded[:], t.Sign)
	if err != nil {
		return false, err
	}

	acc := crypto.PubkeyToAddress(*pubkey)
	return acc.Hex() == t.From.Hex(), nil
}
