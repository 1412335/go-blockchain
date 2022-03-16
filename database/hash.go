package database

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

type Hash [32]byte

func (h Hash) MarshalText() ([]byte, error) {
	// return []byte(base64.StdEncoding.EncodeToString(h[:])), nil
	return []byte(hex.EncodeToString(h[:])), nil
}

func (h *Hash) UnmarshalText(data []byte) error {
	// _, err := base64.StdEncoding.Decode(h[:], data)
	_, err := hex.Decode(h[:], data)
	return err
}

func (h *Hash) Hex() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) IsEmpty() bool {
	emptyHash := Hash{}
	return bytes.Equal(h[:], emptyHash[:])
}

func (h Hash) IsBlockHashValid() bool {
	return fmt.Sprintf("%x", h[:3]) == "000000" && fmt.Sprintf("%x", h[3]) != "0"
}
