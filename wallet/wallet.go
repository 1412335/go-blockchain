package wallet

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/1412335/the-blockchain-bar/database"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const keystoreDirName = "keystore"
const AndrejAccount = "0xf57913DB69e172c0aD5018Fb0CEBf63308B2B8D7"
const BabayagaAccount = "0xca22E5F9C5ae099f64991AB356826C4d52554bF8"
const CaesarAccount = "0xbd75C01F9b4df2DCC34e48f01ae54652F955a42e"

func GetKeystoreDirPath(dir string) string {
	return filepath.Join(dir, keystoreDirName)
}

func NewKeyStoreAccount(dir, pwd string) (database.Account, error) {
	ks := keystore.NewKeyStore(GetKeystoreDirPath(dir), keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(pwd)
	if err != nil {
		return database.Account{}, fmt.Errorf("error creating account: %v", err)
	}
	return database.NewAccount(acc.Address.Hex()), nil
}

func SignTxWithKeystoreAccount(tx database.TX, account database.Account, pwd string, dir string) (database.SignedTx, error) {
	ks := keystore.NewKeyStore(dir, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.Find(accounts.Account{Address: common.Address(account)})
	if err != nil {
		return database.SignedTx{}, err
	}

	ksAccountJSON, err := ioutil.ReadFile(acc.URL.Path)
	if err != nil {
		return database.SignedTx{}, err
	}

	key, err := keystore.DecryptKey(ksAccountJSON, pwd)
	if err != nil {
		return database.SignedTx{}, err
	}

	return SignTx(tx, key.PrivateKey)
}

func SignTx(tx database.TX, privkey *ecdsa.PrivateKey) (database.SignedTx, error) {
	txEncoded, err := tx.Encode()
	if err != nil {
		return database.SignedTx{}, err
	}

	sign, err := Sign(txEncoded, privkey)
	if err != nil {
		return database.SignedTx{}, err
	}

	return database.SignedTx{
		TX:   tx,
		Sign: sign,
	}, nil
}

func Sign(msg []byte, privkey *ecdsa.PrivateKey) ([]byte, error) {
	msgHash := crypto.Keccak256(msg)

	sign, err := crypto.Sign(msgHash, privkey)
	if err != nil {
		return nil, err
	}

	if len(sign) != crypto.SignatureLength {
		return nil, fmt.Errorf("wrong size signature: expected %d bytes, got %d", crypto.SignatureLength, len(sign))
	}

	return sign, nil
}

func Verify(msg, sign []byte) (*ecdsa.PublicKey, error) {
	msgHash := crypto.Keccak256(msg)

	recoveredPublicKey, err := crypto.SigToPub(msgHash, sign)
	if err != nil {
		return nil, fmt.Errorf("unable to verify signature: %v", err)
	}

	return recoveredPublicKey, nil
}

func PublicKeyToAccount(pubkey ecdsa.PublicKey) database.Account {
	// pubkeyBytes := elliptic.Marshal(crypto.S256(), pubkey.X, pubkey.Y)
	// pubkeyBytesHash := crypto.Keccak256(pubkeyBytes[1:])
	// account := common.BytesToAddress(pubkeyBytesHash[12:])
	return database.Account(crypto.PubkeyToAddress(pubkey))
}
