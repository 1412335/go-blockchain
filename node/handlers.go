package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
)

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

	tx := database.NewTX(database.Account(txAddReq.From), database.Account(txAddReq.To), txAddReq.Value, txAddReq.Data)

	block := database.NewBlock(n.state.LatestBlockHash(), n.state.NextBlockNumber(), uint64(time.Now().Unix()), []database.TX{tx})

	hash, err := n.state.AddBlock(block)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeResponse(w, TxAddRes{hash})
}

func nodeStatusHandler(w http.ResponseWriter, _ *http.Request, n *Node) {
	res := StatusRes{
		Hash:       n.state.LatestBlockHash(),
		Number:     n.state.LatestBlock().Header.Number,
		KnownPeers: n.knownPeers,
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
