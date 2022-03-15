package database

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type State struct {
	Balances  map[Account]uint `json:"balances"`
	txMempool []TX

	dbFile *os.File

	latestBlock     Block
	latestBlockHash Hash
	hasGenesisBlock bool
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

	state := &State{balances, make([]TX, 0), f, Block{}, Hash{}, false}

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

		state.latestBlock = blockFS.Block
		state.latestBlockHash = blockFS.BlockHash
		state.hasGenesisBlock = true
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

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.copy()

	nextExpectedBlockNumber := pendingState.NextBlockNumber()
	if pendingState.hasGenesisBlock {
		if b.Header.Number != nextExpectedBlockNumber {
			return Hash{}, fmt.Errorf("expected block number %d, got %d", nextExpectedBlockNumber, b.Header.Number)
		}
		if pendingState.latestBlock.Header.Number > 0 && !bytes.Equal(pendingState.latestBlockHash[:], b.Header.Parent[:]) {
			return Hash{}, fmt.Errorf("expected block hash %d, got %d", pendingState.latestBlockHash[:], b.Header.Parent[:])
		}
	}

	for _, tx := range b.TXs {
		if err := pendingState.apply(tx); err != nil {
			return Hash{}, err
		}
	}

	hash, err := b.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFS := BlockFS{hash, b}
	blockFSJSON, err := json.Marshal(blockFS)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("Persist new block to disk\n")
	fmt.Printf("\t%s\n", blockFSJSON)
	if _, err := s.dbFile.Write(append(blockFSJSON, '\n')); err != nil {
		return Hash{}, err
	}

	s.Balances = pendingState.Balances
	s.latestBlock = b
	s.latestBlockHash = hash
	s.hasGenesisBlock = true

	fmt.Printf("Block number: %d", s.latestBlock.Header.Number)

	return hash, nil
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

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) NextBlockNumber() uint64 {
	if !s.hasGenesisBlock {
		return 0
	}
	return s.latestBlock.Header.Number + 1
}

func (s *State) Persist() (Hash, error) {
	block := NewBlock(s.latestBlockHash, s.NextBlockNumber(), uint64(time.Now().Unix()), s.txMempool)
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

func (s *State) GetBlocksAfter(hash Hash) ([]Block, error) {
	currentOffset, err := s.dbFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	// re-read the whole file from the first byte
	if _, err := s.dbFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	blocks := []Block{}
	startCollect := false

	if hash.IsEmpty() {
		startCollect = true
	}

	scanner := bufio.NewScanner(s.dbFile)

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFS BlockFS
		if err := json.Unmarshal(scanner.Bytes(), &blockFS); err != nil {
			return nil, err
		}

		if startCollect {
			blocks = append(blocks, blockFS.Block)
			continue
		}

		if bytes.Equal(hash[:], blockFS.BlockHash[:]) {
			startCollect = true
		}
	}

	if _, err := s.dbFile.Seek(currentOffset, io.SeekStart); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (s *State) copy() *State {
	cp := &State{}

	cp.Balances = make(map[Account]uint)
	for accout, balance := range s.Balances {
		cp.Balances[accout] = balance
	}

	cp.txMempool = make([]TX, len(s.txMempool))
	cp.txMempool = append(cp.txMempool, s.txMempool...)

	cp.latestBlock = s.latestBlock
	cp.latestBlockHash = s.latestBlockHash
	cp.hasGenesisBlock = s.hasGenesisBlock

	return cp
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
