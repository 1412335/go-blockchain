package database

import (
	"crypto/sha256"
	"encoding/json"
	"time"
)

type TX struct {
	From  Account `json:"from"`
	To    Account `json:"to"`
	Value uint    `json:"value"`
	Data  string  `json:"data"`
	Time  uint64  `json:"time"`
}

func NewTX(from string, to string, value uint, data string) TX {
	return TX{NewAccount(from), NewAccount(to), value, data, uint64(time.Now().Unix())}
}

func (tx *TX) IsReward() bool {
	return tx.Data == "reward"
}

func (tx *TX) Hash() (Hash, error) {
	txJSON, err := json.Marshal(tx)
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(txJSON), nil
}
