package node

import (
	"encoding/json"
	"net/http"
)

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
