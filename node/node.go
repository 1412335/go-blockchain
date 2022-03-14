package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/1412335/the-blockchain-bar/database"
)

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
	Hash       database.Hash `json:"block_hash"`
	Number     uint64        `json:"block_number"`
	KnownPeers []PeerNode    `json:"known_peers"`
}

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`
	IsActive    bool   `json:"is_active"`
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, isActive bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, isActive}
}

type Node struct {
	dataDir string
	ip      string
	port    uint64

	state *database.State

	knownPeers []PeerNode
}

func New(dataDir string, ip string, port uint64, bootstrap PeerNode) *Node {
	return &Node{
		dataDir:    dataDir,
		ip:         ip,
		port:       port,
		knownPeers: []PeerNode{bootstrap},
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

	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)
	})

	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		addTransactionHandler(w, r, state)
	})

	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		nodeStatusHandler(w, r, n)
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", n.port), nil)
}

func listBalancesHandler(w http.ResponseWriter, _ *http.Request, state *database.State) {
	writeResponse(w, BalancesRes{
		Hash:     state.LatestBlockHash(),
		Balances: state.Balances,
	})
}

func addTransactionHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	reqBodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	defer r.Body.Close()

	var txAddReq TxAddReq
	if err = json.Unmarshal(reqBodyJSON, &txAddReq); err != nil {
		writeErrorResponse(w, err)
		return
	}

	tx := database.NewTX(database.Account(txAddReq.From), database.Account(txAddReq.To), txAddReq.Value, txAddReq.Data)

	if err = state.AddTx(tx); err != nil {
		writeErrorResponse(w, err)
		return
	}

	hash, err := state.Persist()
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeResponse(w, TxAddRes{hash})
}

func nodeStatusHandler(w http.ResponseWriter, _ *http.Request, n *Node) {
	writeResponse(w, StatusRes{
		Hash:       n.state.LatestBlockHash(),
		Number:     n.state.LatestBlock().Header.Number,
		KnownPeers: n.knownPeers,
	})
}

func writeResponse(w http.ResponseWriter, data interface{}) {
	content, err := json.Marshal(data)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func writeErrorResponse(w http.ResponseWriter, err error) {
	errJSON, _ := json.Marshal(ErrRes{err.Error()})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(errJSON)
}
