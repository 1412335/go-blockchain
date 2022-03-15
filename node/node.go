package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
)

const endpointStatus = "/node/status"
const endpointAddPeer = "/node/peer"
const endpointFetchBlocks = "/node/blocks"

type ErrRes struct {
	Error string `json:"error"`
}

type BalancesRes struct {
	Hash     database.Hash             `json:"hash"`
	Balances map[database.Account]uint `json:"balances"`
}

type TxAddReq struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value uint   `json:"value"`
	Data  string `json:"data"`
}

type TxAddRes struct {
	Hash database.Hash `json:"block_hash"`
}

type StatusRes struct {
	Hash       database.Hash       `json:"block_hash"`
	Number     uint64              `json:"block_number"`
	KnownPeers map[string]PeerNode `json:"known_peers"`
}

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`
	connected   bool
}

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type FetchBlocksRes struct {
	Blocks []database.Block `json:"blocks"`
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
}

func New(dataDir string, ip string, port uint64, bootstrap PeerNode) *Node {
	return &Node{
		dataDir: dataDir,
		ip:      ip,
		port:    port,
		knownPeers: map[string]PeerNode{
			bootstrap.TCPAddress(): bootstrap,
		},
	}
}

func (n *Node) Run() error {
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

	ctx := context.Background()
	go n.sync(ctx)

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

	return http.ListenAndServe(fmt.Sprintf(":%d", n.port), nil)
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TCPAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TCPAddress())
}

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(45 * time.Second)

	for {
		select {
		case <-ticker.C:
			fmt.Println("Searching for new Peer and Block...")

			n.fetchNewBlocksAndPeers(ctx)

		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) fetchNewBlocksAndPeers(ctx context.Context) {
	for _, knownPeer := range n.knownPeers {
		if knownPeer.IP == n.ip && knownPeer.Port == n.port {
			continue
		}

		fmt.Printf("Searching for new Peers and their Blocks and Peers: '%s'\n", knownPeer.TCPAddress())

		status, err := queryPeerStatus(ctx, knownPeer)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("Peer '%s' was removed from KnownPeers\n", knownPeer.TCPAddress())

			n.RemovePeer(knownPeer)
			continue
		}

		if err := n.joinKnownPeers(ctx, knownPeer); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if err := n.syncBlocks(ctx, knownPeer, status); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		n.syncKnownPeers(status.KnownPeers)
	}
}

// Add peer to node.knowPeers
func (n *Node) joinKnownPeers(ctx context.Context, peer PeerNode) error {
	if peer.connected {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s%s?ip=%s&port=%d", peer.TCPAddress(), endpointAddPeer, n.ip, n.port), nil)
	if err != nil {
		return err
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	rBodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	addPeerRes := AddPeerRes{}
	if err := json.Unmarshal(rBodyJSON, &addPeerRes); err != nil {
		return err
	}
	if addPeerRes.Error != "" {
		return fmt.Errorf(addPeerRes.Error)
	}

	peer.connected = addPeerRes.Success
	n.AddPeer(peer)

	if !addPeerRes.Success {
		return fmt.Errorf("unable to join KnownPeers of '%s'", peer.TCPAddress())
	}
	return nil
}

// Fetch blocks from peer
func (n *Node) syncBlocks(ctx context.Context, peer PeerNode, status StatusRes) error {
	localBlockNumber := n.state.LatestBlock().Header.Number

	if status.Hash.IsEmpty() {
		return nil
	}

	if status.Number < localBlockNumber {
		return nil
	}

	if status.Number == 0 && !n.state.LatestBlockHash().IsEmpty() {
		return nil
	}

	newBlocksCount := status.Number - localBlockNumber
	if localBlockNumber == 0 && status.Number == 0 {
		newBlocksCount = 1
	}
	fmt.Printf("Found %d new blocks from Peer %s\n", newBlocksCount, peer.TCPAddress())

	if newBlocksCount == 0 {
		return nil
	}

	blocks, err := fetchBlocksFromPeer(ctx, peer, n.state.LatestBlockHash())
	if err != nil {
		return err
	}

	for _, block := range blocks {
		if _, err := n.state.AddBlock(block); err != nil {
			return err
		}
	}
	return nil
}

// Sync node.knownPeers with peer.knownPeers
func (n *Node) syncKnownPeers(knownPeers map[string]PeerNode) {
	for _, peer := range knownPeers {
		if peer.IP == n.ip && peer.Port == n.port {
			continue
		}
		if _, isKnown := n.knownPeers[peer.TCPAddress()]; !isKnown {
			fmt.Printf("Found new Peer: %s\n", peer.TCPAddress())
			n.AddPeer(peer)
		}
	}
}

func queryPeerStatus(ctx context.Context, peer PeerNode) (StatusRes, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s%s", peer.TCPAddress(), endpointStatus), nil)
	if err != nil {
		return StatusRes{}, err
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return StatusRes{}, err
	}

	rBodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return StatusRes{}, err
	}
	defer r.Body.Close()

	var statusRes StatusRes
	if err = json.Unmarshal(rBodyJSON, &statusRes); err != nil {
		return StatusRes{}, err
	}
	return statusRes, nil
}

func fetchBlocksFromPeer(ctx context.Context, peer PeerNode, hash database.Hash) ([]database.Block, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s%s?hash=%x", peer.TCPAddress(), endpointFetchBlocks, hash), nil)
	if err != nil {
		return nil, err
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	rBodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var statusRes FetchBlocksRes
	if err = json.Unmarshal(rBodyJSON, &statusRes); err != nil {
		return nil, err
	}
	return statusRes.Blocks, nil
}
