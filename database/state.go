package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type State struct {
	Balances        map[Account]uint `json:"balances"`
	txMempool       []TX
	latestBlockHash Hash

	dbFile *os.File
}

func NewStateFromDisk(dir string) (*State, error) {
	if err := initDataDirIfNotExists(dir); err != nil {
		return nil, err
	}

	genesisPath := getGenesisJSONFilePath(dir)
	genesis, err := loadGenesis(genesisPath)
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint, len(genesis.Balances))
	for account, balance := range genesis.Balances {
		balances[account] = balance
	}

	f, err := os.OpenFile(getBlocksDBFilePath(dir), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	state := &State{balances, make([]TX, 0), Hash{}, f}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFS BlockFS
		if err := json.Unmarshal(scanner.Bytes(), &blockFS); err != nil {
			return nil, err
		}

		if err := state.applyBlock(blockFS.Block); err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFS.BlockHash
	}

	return state, nil
}

func (s *State) AddTx(tx TX) error {
	if err := s.apply(tx); err != nil {
		return err
	}
	s.txMempool = append(s.txMempool, tx)
	return nil
}

func (s *State) AddBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.AddTx(tx); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) apply(tx TX) error {
	if tx.IsReward() {
		s.Balances[tx.To] += tx.Value
		return nil
	}

	if s.Balances[tx.From] < tx.Value {
		return fmt.Errorf("not enough balance")
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value

	return nil
}

func (s *State) applyBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.apply(tx); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) Persist() (Hash, error) {
	block := NewBlock(s.latestBlockHash, uint64(time.Now().Unix()), s.txMempool)
	blockHash, err := block.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFS := BlockFS{blockHash, block}
	blockFSJSON, err := json.Marshal(blockFS)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("Persist new block to disk\n")
	fmt.Printf("\t%s\n", blockFSJSON)
	if _, err := s.dbFile.Write(append(blockFSJSON, '\n')); err != nil {
		return Hash{}, err
	}

	s.latestBlockHash = blockHash
	fmt.Printf("New DB hash: %x\n", s.latestBlockHash)

	s.txMempool = []TX{}

	return s.latestBlockHash, nil
}

func (s *State) Close() error {
	return s.dbFile.Close()
}

// func (s *State) doSnapshot() error {
// 	// re-read the whole file from the first byte
// 	if _, err := s.dbFile.Seek(0, io.SeekStart); err != nil {
// 		return err
// 	}

// 	txData, err := ioutil.ReadAll(s.dbFile)
// 	if err != nil {
// 		return err
// 	}

// 	s.latestBlockHash = sha256.Sum256(txData)

// 	return nil
// }
