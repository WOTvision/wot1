package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"golang.org/x/crypto/ed25519"
)

// WalletFlagAES256Keys is the flag which, if present, specifies the keys are encrypted with AES256-GCM
const WalletFlagAES256Keys = "aes256_keys"

// DefaultWalletFilename is the default filename for the wallet file
const DefaultWalletFilename = "wallet.json"

// Wallet is the serialised (exportable / importable) representation of a user's wallet.
type Wallet struct {
	Name          string      `json:"name"`
	Version       int         `json:"version"`
	Flags         []string    `json:"flags"` // e.g. "aes256_keys"
	CachedBalance int64       `json:"cached_balance"`
	Keys          []WalletKey `json:"keys"`
}

// WalletKey is a record of a public-private keypair
type WalletKey struct {
	Name          string   `json:"name"` // Unique among the keys in this wallet
	Private       string   `json:"private"`
	Public        string   `json:"public"`
	Flags         []string `json:"flags"`
	CachedBalance int64    `json:"cached_balance"`

	pub  ed25519.PublicKey  // public key, internal representation
	priv ed25519.PrivateKey // private key, internal representation
}

// The current wallet, a global variable
var currentWallet = Wallet{}

func (w *Wallet) createKey(name string, password string) error {
	if !inStringSlice(WalletFlagAES256Keys, w.Flags) {
		return fmt.Errorf("Need %s", WalletFlagAES256Keys)
	}
	var err error

	passwordHash := sha256.Sum256([]byte(password))

	wk := WalletKey{Name: name, Flags: []string{}}
	wk.pub, wk.priv, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	aesBlock, err := aes.NewCipher(passwordHash[:])
	if err != nil {
		return err
	}
	aesStream, err := cipher.NewGCM(aesBlock)
	if err != nil {
		log.Panic("Cannot create AES-GCM")
	}
	nonce := make([]byte, aesStream.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Panic("Cannot read rand.Reader")
	}
	privEnc := aesStream.Seal(nil, nonce, wk.priv, nil)

	wk.Private = base64.StdEncoding.EncodeToString(nonce) + "." + base64.StdEncoding.EncodeToString(privEnc)
	wk.Public = base64.StdEncoding.EncodeToString(wk.pub)

	w.Keys = append(w.Keys, wk)
	return nil
}

// Save saves the wallet to the given filename in JSON format
func (w *Wallet) Save(filename string) error {
	return ioutil.WriteFile(filename, jsonifyWhateverToBytes(*w), 0600)
}

// LoadWallet loads a wallet from a JSON file
func LoadWallet(filename, password string) (*Wallet, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	w := Wallet{}
	err = json.Unmarshal(data, &w)
	if err != nil {
		return nil, err
	}
	if !inStringSlice(WalletFlagAES256Keys, w.Flags) {
		return nil, fmt.Errorf("Only supporting wallets with %s", WalletFlagAES256Keys)
	}

	passwordHash := sha256.Sum256([]byte(password))
	for ki := range w.Keys {

		if password != "" { // Decrypt the private key if the password is non-empty
			privB64 := strings.Split(w.Keys[ki].Private, ".")
			nonce, err := base64.StdEncoding.DecodeString(privB64[0])
			if err != nil {
				return nil, err
			}
			privEnc, err := base64.StdEncoding.DecodeString(privB64[1])
			if err != nil {
				return nil, err
			}

			aesBlock, err := aes.NewCipher(passwordHash[:])
			if err != nil {
				return nil, err
			}
			aesStream, err := cipher.NewGCM(aesBlock)
			if err != nil {
				return nil, err
			}
			w.Keys[ki].priv, err = aesStream.Open(nil, nonce, privEnc, nil)
			if err != nil {
				return nil, err
			}
		}

		w.Keys[ki].pub, err = base64.StdEncoding.DecodeString(w.Keys[ki].Public)
		if err != nil {
			return nil, err
		}
	}

	return &w, nil
}

func initWallet() {
	w, err := LoadWallet(*walletFileName, "")
	if err != nil {
		log.Println("Cannot load", *walletFileName)
		return
	}
	currentWallet = *w
	for _, key := range w.Keys {
		if key.priv != nil {
			log.Panic("Attempt to load locked wallet resulted in unlocked wallet.")
		}
	}
	log.Println("Loaded", *walletFileName, "(locked)")
}
