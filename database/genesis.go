package database

import (
	"encoding/json"
	"io/ioutil"
)

var genesisJSON = `
{
  "genesis_time": "2019-03-18T00:00:00.000000000Z",
  "chain_id": "the-blockchain-bar-ledger",
  "balances": {
    "0xf57913DB69e172c0aD5018Fb0CEBf63308B2B8D7": 1000000
  }
}`

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

func writeGenesisToDisk(genPath string) error {
	//nolint:gosec
	ioutil.WriteFile(genPath, []byte(genesisJSON), 0644)
	return nil
}
