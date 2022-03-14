package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/1412335/the-blockchain-bar/database"
)

const httpPort = 8080

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

func Run(dir string) error {
	state, err := database.NewStateFromDisk(dir)
	if err != nil {
		return err
	}
	defer state.Close()

	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)
	})

	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		addTransactionHandler(w, r, state)
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
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
