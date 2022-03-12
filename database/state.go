package database

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
)

type Snapshot [32]byte

type Block struct {
	Header BlockHeader
	TXs []TX
}

func (b *Block) Hash() (Snapshot, error) {
	blockJSON, err := json.Marshal(b)
	if err != nil {
		return Snapshot{}, err
	}
	return sha256.Sum256(blockJSON), nil
}

type BlockHeader struct {
	Parent Snapshot
	Time uint64
}

type State struct {
	Balances  map[Account]uint `json:"balances"`
	txMempool []TX

	dbFile *os.File

	snapshot Snapshot
}

func NewStateFromDisk() (*State, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	genesisPath := path.Join(cwd, "database", "genesis.json")
	genesis, err := loadGenesis(genesisPath)
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint, len(genesis.Balances))
	for account, balance := range genesis.Balances {
		balances[account] = balance
	}

	f, err := os.OpenFile(path.Join(cwd, "database", "tx.db"), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	state := &State{balances, make([]TX, 0), f, Snapshot{}}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var tx TX
		if err := json.Unmarshal(scanner.Bytes(), &tx); err != nil {
			return nil, err
		}

		if err := state.apply(tx); err != nil {
			return nil, err
		}
	}

	return state, nil
}

func (s *State) Add(tx TX) error {
	if err := s.apply(tx); err != nil {
		return err
	}
	s.txMempool = append(s.txMempool, tx)
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

func (s *State) Persist() (Snapshot, error) {
	mempoolCp := make([]TX, len(s.txMempool))
	copy(mempoolCp, s.txMempool)

	for i := range mempoolCp {
		txJSON, err := json.Marshal(mempoolCp[i])
		if err != nil {
			return Snapshot{}, err
		}

		fmt.Printf("Persist new tx to disk\n")
		fmt.Printf("%s\n", txJSON)
		if _, err := s.dbFile.Write(append(txJSON, '\n')); err != nil {
			return Snapshot{}, err
		}

		if err := s.doSnapshot(); err != nil {
			return Snapshot{}, err
		}
		fmt.Printf("New DB snapshot: %x\n", s.snapshot)

		s.txMempool = s.txMempool[1:]
	}

	return s.snapshot, nil
}

func (s *State) Close() error {
	return s.dbFile.Close()
}

func (s *State) doSnapshot() error {
	// re-read the whole file from the first byte
	if _, err := s.dbFile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	txData, err := ioutil.ReadAll(s.dbFile)
	if err != nil {
		return err
	}

	s.snapshot = sha256.Sum256(txData)

	return nil
}

func (s *State) LatestSnapshot() Snapshot {
	return s.snapshot
}
