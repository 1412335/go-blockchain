package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
)

const endpointStatus = "/node/status"
const endpointAddPeer = "/node/peer"
const endpointFetchBlocks = "/node/blocks"

const miningIntervalSecs = 10

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`
	connected   bool
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, connected bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, connected}
}

func (p *PeerNode) TCPAddress() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

type Node struct {
	dataDir string
	ip      string
	port    uint64

	state *database.State

	knownPeers map[string]PeerNode

	archiveTxs     map[string]database.TX
	pendingTxs     map[string]database.TX
	isMining       bool
	miner          database.Account
	newSyncedBlock chan database.Block
}

func New(dataDir string, ip string, port uint64, miner database.Account, bootstrap PeerNode) *Node {
	return &Node{
		dataDir: dataDir,
		ip:      ip,
		port:    port,
		knownPeers: map[string]PeerNode{
			bootstrap.TCPAddress(): bootstrap,
		},
		archiveTxs:     make(map[string]database.TX),
		pendingTxs:     make(map[string]database.TX),
		isMining:       false,
		miner:          miner,
		newSyncedBlock: make(chan database.Block),
	}
}

func (n *Node) Run(ctx context.Context) error {
	fmt.Printf("Listening on HTTP port: %d\n", n.port)

	state, err := database.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()

	n.state = state

	fmt.Println("Blockchain state:")
	fmt.Printf("	- height: %d\n", n.state.LatestBlock().Header.Number)
	fmt.Printf("	- hash: %x\n", n.state.LatestBlockHash())

	go n.sync(ctx)
	go n.mine(ctx)

	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, n)
	})

	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		addTransactionHandler(w, r, n)
	})

	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		nodeStatusHandler(w, r, n)
	})

	http.HandleFunc(endpointAddPeer, func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	http.HandleFunc(endpointFetchBlocks, func(w http.ResponseWriter, r *http.Request) {
		fetchBlocksHandler(w, r, n)
	})

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", n.port),
	}

	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()

	err = server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TCPAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TCPAddress())
}

func (n *Node) AddPendingTX(tx database.TX, peer PeerNode) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	txJSON, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	_, isPending := n.pendingTxs[txHash.Hex()]
	_, isArchived := n.archiveTxs[txHash.Hex()]

	if !isPending && !isArchived {
		fmt.Printf("Added Pending TX %s from Peer %s\n", txJSON, peer.TCPAddress())
		n.pendingTxs[txHash.Hex()] = tx
	}
	return nil
}

func (n *Node) mine(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * miningIntervalSecs)

	var miningCtx context.Context
	var miningCancel context.CancelFunc

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTxs) > 0 && !n.isMining {
					n.isMining = true

					miningCtx, miningCancel = context.WithCancel(ctx)
					if err := n.miningPendingTxs(miningCtx); err != nil {
						fmt.Printf("Error: %v\n", err)
					}

					n.isMining = false
				}
			}()
		case block := <-n.newSyncedBlock:
			if n.isMining {
				hash, err := block.Hash()
				if err != nil {
					return err
				}
				fmt.Printf("Miner '%s' mined next Block '%x' faster\n", block.Header.Miner, hash)

				if err := n.removeMinedPendingTXs(block); err != nil {
					return err
				}
				miningCancel()
			}
		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) miningPendingTxs(ctx context.Context) error {
	var pendingTxs []database.TX
	for _, tx := range n.pendingTxs {
		pendingTxs = append(pendingTxs, tx)
	}

	pb := NewPendingBlock(n.state.LatestBlockHash(), n.state.NextBlockNumber(), n.miner, pendingTxs)

	minedBlock, err := Mine(ctx, pb)
	if err != nil {
		return err
	}

	if err := n.removeMinedPendingTXs(minedBlock); err != nil {
		return err
	}

	if _, err := n.state.AddBlock(minedBlock); err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTXs(block database.Block) error {
	if len(n.pendingTxs) == 0 || len(block.TXs) == 0 {
		return nil
	}

	for _, tx := range block.TXs {
		txHash, err := tx.Hash()
		if err != nil {
			return err
		}

		if _, exists := n.pendingTxs[txHash.Hex()]; exists {
			delete(n.pendingTxs, txHash.Hex())

			fmt.Printf("\t-archiving mined TX: %s\n", txHash.Hex())
			n.archiveTxs[txHash.Hex()] = tx
		}
	}
	return nil
}
