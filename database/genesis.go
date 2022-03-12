package database

import (
	"encoding/json"
	"io/ioutil"
)

type genesis struct {
	Balances map[Account]uint `json:"balances"`
}

func loadGenesis(path string) (genesis, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return genesis{}, err
	}

	var loadedGenesis genesis
	if err := json.Unmarshal(contents, &loadedGenesis); err != nil {
		return genesis{}, err
	}
	return loadedGenesis, nil
}
