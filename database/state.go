package database

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const BlockReward = 100

type State struct {
	Balances  map[Account]uint `json:"balances"`
	txMempool []SignedTx

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

	state := &State{balances, make([]SignedTx, 0), f, Block{}, Hash{}, false}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFS BlockFS
		if err := json.Unmarshal(scanner.Bytes(), &blockFS); err != nil {
			return nil, err
		}

		if err := applyBlock(state, blockFS.Block); err != nil {
			return nil, err
		}

		state.latestBlock = blockFS.Block
		state.latestBlockHash = blockFS.BlockHash
		state.hasGenesisBlock = true
	}

	return state, nil
}

func (s *State) AddTx(tx SignedTx) error {
	if err := s.apply(tx); err != nil {
		return err
	}
	s.txMempool = append(s.txMempool, tx)
	return nil
}

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.copy()

	if err := applyBlock(pendingState, b); err != nil {
		return Hash{}, err
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

	return hash, nil
}

func (s *State) apply(tx SignedTx) error {
	isAuth, err := tx.IsAuthentic()
	if err != nil {
		return err
	}
	if !isAuth {
		return fmt.Errorf("wrong TX. Sender '%s' is forged", tx.From.Hex())
	}

	if tx.IsReward() {
		s.Balances[tx.To] += tx.Value
		return nil
	}

	if s.Balances[tx.From] < tx.Value {
		return fmt.Errorf("wrong TX. Sender %s balance is %d, but cost is %d", tx.From.Hex(), s.Balances[tx.From], tx.Value)
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value

	return nil
}

func applyBlock(state *State, b Block) error {
	nextExpectedBlockNumber := state.NextBlockNumber()
	if state.hasGenesisBlock {
		if b.Header.Number != nextExpectedBlockNumber {
			return fmt.Errorf("expected block number %d, got %d", nextExpectedBlockNumber, b.Header.Number)
		}
		if state.latestBlock.Header.Number > 0 && !bytes.Equal(state.latestBlockHash[:], b.Header.Parent[:]) {
			return fmt.Errorf("expected block hash %d, got %d", state.latestBlockHash[:], b.Header.Parent[:])
		}
	}

	hash, err := b.Hash()
	if err != nil {
		return err
	}

	if !hash.IsBlockHashValid() {
		return fmt.Errorf("invalid block hash %x", hash)
	}

	for _, tx := range b.TXs {
		if err := state.apply(tx); err != nil {
			return err
		}
	}

	state.Balances[b.Header.Miner] += BlockReward

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

// func (s *State) Persist() (Hash, error) {
// 	block := NewBlock(s.latestBlockHash, s.NextBlockNumber(), uint64(time.Now().Unix()), s.txMempool)
// 	blockHash, err := block.Hash()
// 	if err != nil {
// 		return Hash{}, err
// 	}

// 	blockFS := BlockFS{blockHash, block}
// 	blockFSJSON, err := json.Marshal(blockFS)
// 	if err != nil {
// 		return Hash{}, err
// 	}

// 	fmt.Printf("Persist new block to disk\n")
// 	fmt.Printf("\t%s\n", blockFSJSON)
// 	if _, err := s.dbFile.Write(append(blockFSJSON, '\n')); err != nil {
// 		return Hash{}, err
// 	}

// 	s.latestBlockHash = blockHash
// 	fmt.Printf("New DB hash: %x\n", s.latestBlockHash)

// 	s.txMempool = []TX{}

// 	return s.latestBlockHash, nil
// }

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

	cp.txMempool = make([]SignedTx, len(s.txMempool))
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
