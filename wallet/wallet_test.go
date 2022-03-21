package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSign(t *testing.T) {
	privkey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	account := PublicKeyToAccount(privkey.PublicKey)

	msg := []byte("message")
	sign, err := Sign(msg, privkey)
	if err != nil {
		t.Fatal(err)
	}

	recoveredPublicKey, err := Verify(msg, sign)
	if err != nil {
		t.Fatal(err)
	}

	recoveredAccount := PublicKeyToAccount(*recoveredPublicKey)

	if account.Hex() != recoveredAccount.Hex() {
		t.Fatalf("msg signed by %s, got %s", account.Hex(), recoveredAccount.Hex())
	}
}
