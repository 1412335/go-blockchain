package database

type Account string

func NewAccount(name string) Account {
	return Account(name)
}

type TX struct {
	From  Account `json:"from"`
	To    Account `json:"to"`
	Value uint    `json:"value"`
	Data  string  `json:"data"`
}

func NewTX(from Account, to Account, value uint, data string) TX {
	return TX{from, to, value, data}
}

func (tx *TX) IsReward() bool {
	return tx.Data == "reward"
}
