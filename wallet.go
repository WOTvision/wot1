package main

import (
	"crypto"
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
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/ed25519"
)

// WalletFlagAES256Keys is the flag which, if present, specifies the keys are encrypted with AES256-GCM
const WalletFlagAES256Keys = "aes256_keys"

const PublicKeyPrefix = 'W'

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
	Name          string    `json:"name"` // Unique among the keys in this wallet
	Private       string    `json:"private"`
	Public        string    `json:"public"`
	Flags         []string  `json:"flags"`
	CachedBalance int64     `json:"cached_balance"`
	CreationTime  time.Time `json:"ctime"`

	pub  ed25519.PublicKey  // public key, internal representation
	priv ed25519.PrivateKey // private key, internal representation, nil if locked/encrypted
}

// The current wallet, a global variable
var currentWallet = Wallet{}
var currentWalletFile string

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

	wk.Private = base64.RawURLEncoding.EncodeToString(nonce) + "." + base64.RawURLEncoding.EncodeToString(privEnc)
	wk.Public = string(PublicKeyPrefix) + base64.RawURLEncoding.EncodeToString(wk.pub)
	wk.CreationTime = time.Now()

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

	for ki := range w.Keys {
		if w.Keys[ki].Public[0] != PublicKeyPrefix {
			return nil, fmt.Errorf("Public key in wallet with invalid prefix '%s'", string(w.Keys[ki].Public[0]))
		}

		if password != "" { // Decrypt the private key if the password is non-empty
			err := w.Keys[ki].UnlockPrivateKey(password)
			if err != nil {
				return nil, err
			}
		}

		w.Keys[ki].pub, err = base64.RawURLEncoding.DecodeString(w.Keys[ki].Public[1:])
		if err != nil {
			return nil, err
		}
	}

	return &w, nil
}

// UnlockPrivateKey unlocks the keypair's private part
func (wk *WalletKey) UnlockPrivateKey(password string) error {
	if wk.priv != nil {
		return fmt.Errorf("Private key already unlocked")
	}
	passwordHash := sha256.Sum256([]byte(password))

	privB64 := strings.Split(wk.Private, ".")
	nonce, err := base64.RawURLEncoding.DecodeString(privB64[0])
	if err != nil {
		return err
	}
	privEnc, err := base64.RawURLEncoding.DecodeString(privB64[1])
	if err != nil {
		return err
	}

	aesBlock, err := aes.NewCipher(passwordHash[:])
	if err != nil {
		return err
	}
	aesStream, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return err
	}
	wk.priv, err = aesStream.Open(nil, nonce, privEnc, nil)
	if err != nil {
		return err
	}
	return nil
}

// SignRaw signs the provided data with the private key and returns raw signed data
func (wk *WalletKey) SignRaw(data []byte) ([]byte, error) {
	if wk.priv == nil {
		return nil, fmt.Errorf("Private key is locked")
	}
	return wk.priv.Sign(rand.Reader, data, crypto.Hash(0))
}

func (ws *WalletKey) VerifyRaw(msg, sig []byte) error {
	if ws.pub == nil {
		return fmt.Errorf("No public key in keypair")
	}
	if len(ws.pub) != ed25519.PublicKeySize {
		return fmt.Errorf("Invalid public key in keypair")
	}
	if ed25519.Verify(ws.pub, msg, sig) {
		return nil
	}
	return fmt.Errorf("Signature verification failed")
}

func DecodePublicKeyString(s string) (*WalletKey, error) {
	if s[0] != PublicKeyPrefix {
		return nil, fmt.Errorf("Invalid public key prefix '%s'", string(s[0]))
	}
	w := WalletKey{Public: s}
	var err error
	w.pub, err = base64.RawURLEncoding.DecodeString(s[1:])
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func initWallet(create bool) {
	wFile := *walletFileName
	if !path.IsAbs(wFile) && wFile[0] != '.' {
		wFile = path.Join(*dataDir, *walletFileName)
	}
	if _, err := os.Stat(wFile); err != nil {
		if create {
			w := Wallet{Name: "default", Flags: []string{WalletFlagAES256Keys}}
			err := w.createKey("default", "password")
			if err != nil {
				log.Fatal("Cannot create key:", err)
			}
			err = w.Save(wFile)
			if err != nil {
				log.Fatal(err)
			}
			log.Println(fmt.Sprintf("Created a key named '%s'", w.Keys[0].Name))
		} else {
			log.Panicln("Can neither load or create wallet", wFile)
		}
	}

	w, err := LoadWallet(wFile, "")
	if err != nil {
		log.Println("Cannot load wallet", wFile, err)
		return
	}
	currentWallet = *w
	for _, key := range w.Keys {
		if key.priv != nil {
			log.Panic("Attempt to load locked wallet resulted in unlocked wallet.")
		}
	}
	currentWalletFile = wFile
	log.Println("Loaded", currentWalletFile)
}
