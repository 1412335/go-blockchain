package database

import (
	"github.com/ethereum/go-ethereum/common"
)

type Account common.Address

func NewAccount(name string) Account {
	return Account(common.HexToAddress(name))
}

func (a Account) MarshalText() ([]byte, error) {
	return []byte(a.Hex()), nil
}

func (a *Account) UnmarshalText(data []byte) error {
	*a = Account(common.HexToAddress(string(data)))
	return nil
}

func (a Account) Hex() string {
	return common.Address(a).Hex()
}
