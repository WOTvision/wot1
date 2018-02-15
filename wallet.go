package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/ed25519"
)

// WalletFlagAES256Keys is the flag which, if present, specifies the keys are encrypted with AES256-GCM
const WalletFlagAES256Keys = "aes256_keys"

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
}

func (w *Wallet) createKey(name string, password string) error {
	if !inStringSlice(WalletFlagAES256Keys, w.Flags) {
		return fmt.Errorf("Need %s", WalletFlagAES256Keys)
	}

	passwordHash := sha256.Sum256([]byte(password))

	wk := WalletKey{Name: name, Flags: []string{}}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
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
	privEnc := aesStream.Seal(nil, nonce, priv, nil)

	wk.Private = base64.StdEncoding.EncodeToString(nonce) + "." + base64.StdEncoding.EncodeToString(privEnc)
	wk.Public = base64.StdEncoding.EncodeToString(pub)

	w.Keys = append(w.Keys, wk)
	return nil
}
