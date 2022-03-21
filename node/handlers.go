package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/1412335/the-blockchain-bar/wallet"
	"github.com/ethereum/go-ethereum/common"
)

type ErrRes struct {
	Error string `json:"error"`
}

type BalancesRes struct {
	Hash     database.Hash             `json:"hash"`
	Balances map[database.Account]uint `json:"balances"`
}

type TxAddReq struct {
	From    string `json:"from"`
	FromPwd string `json:"from_pwd"`
	To      string `json:"to"`
	Value   uint   `json:"value"`
	Data    string `json:"data"`
}

type TxAddRes struct {
	// Hash    database.Hash `json:"block_hash"`
	Success bool `json:"success"`
}

type StatusRes struct {
	Hash       database.Hash       `json:"block_hash"`
	Number     uint64              `json:"block_number"`
	KnownPeers map[string]PeerNode `json:"known_peers"`

	PendingTxs []database.SignedTx `json:"pending_txs"`
}

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type FetchBlocksRes struct {
	Blocks []database.Block `json:"blocks"`
}

func listBalancesHandler(w http.ResponseWriter, _ *http.Request, n *Node) {
	writeResponse(w, BalancesRes{
		Hash:     n.state.LatestBlockHash(),
		Balances: n.state.Balances,
	})
}

func addTransactionHandler(w http.ResponseWriter, r *http.Request, n *Node) {
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

	from := database.NewAccount(txAddReq.From)
	if from.Hex() == common.HexToAddress("").Hex() {
		writeErrorResponse(w, fmt.Errorf("from is invalid %s", from.Hex()))
		return
	}

	if txAddReq.FromPwd == "" {
		writeErrorResponse(w, fmt.Errorf("from password is missing"))
		return
	}

	tx := database.NewTX(txAddReq.From, txAddReq.To, txAddReq.Value, txAddReq.Data)

	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, from, txAddReq.FromPwd, wallet.GetKeystoreDirPath(n.dataDir))
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	// txHash, err := tx.Hash()
	// if err != nil {
	// 	writeErrorResponse(w, err)
	// 	return
	// }
	// n.pendingTxs[txHash.Hex()] = signedTx
	if err := n.AddPendingTX(signedTx, NewPeerNode(n.ip, n.port, false, true)); err != nil {
		writeErrorResponse(w, err)
		return
	}

	// nonce, err := database.RandomNonce()
	// if err != nil {
	// 	writeErrorResponse(w, err)
	// 	return
	// }

	// block := database.NewBlock(n.state.LatestBlockHash(), n.state.NextBlockNumber(), uint64(time.Now().Unix()), nonce, []database.TX{tx})

	// hash, err := n.state.AddBlock(block)
	// if err != nil {
	// 	writeErrorResponse(w, err)
	// 	return
	// }

	writeResponse(w, TxAddRes{true})
}

func nodeStatusHandler(w http.ResponseWriter, _ *http.Request, n *Node) {
	var pendingTxs []database.SignedTx
	for _, tx := range n.pendingTxs {
		pendingTxs = append(pendingTxs, tx)
	}

	res := StatusRes{
		Hash:       n.state.LatestBlockHash(),
		Number:     n.state.LatestBlock().Header.Number,
		KnownPeers: n.knownPeers,
		PendingTxs: pendingTxs,
	}
	writeResponse(w, res)
}

func addPeerHandler(w http.ResponseWriter, r *http.Request, n *Node) {
	ip := r.URL.Query().Get("ip")
	portRaw := r.URL.Query().Get("port")

	port, err := strconv.ParseUint(portRaw, 10, 32)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	peer := NewPeerNode(ip, port, false, true)
	n.AddPeer(peer)
	fmt.Printf("Peer '%s' was added into KnownPeers\n", peer.TCPAddress())

	writeResponse(w, AddPeerRes{true, ""})
}

func fetchBlocksHandler(w http.ResponseWriter, r *http.Request, n *Node) {
	hashRaw := r.URL.Query().Get("hash")

	hash := database.Hash{}
	if err := hash.UnmarshalText([]byte(hashRaw)); err != nil {
		writeErrorResponse(w, err)
		return
	}

	blocks, err := n.state.GetBlocksAfter(hash)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeResponse(w, FetchBlocksRes{blocks})
}
