package main

import (
	"crypto/sha256"
)

const CurrentTxVersion = 1

// A transaction output
type TxOutput struct {
	PubKey string `json:"k"`
	Amount uint64 `json:"a"`
	Nonce  uint64 `json:"n"` // new nonce after this tx is executed
	Data   string `json:"d"` // optional
}

// Tx is a transaction. It's usually kept as a JSON-serialized string in a block
type Tx struct {
	Version        uint          `json:"v"`
	SigningPubKey  string        `json:"k"`
	Flags          []string      `json:"f"` // "coinbase"
	Outputs        []TxOutput    `json:"o"`
	MinerFeeAmount uint          `json:"m"`
	Data           PublishedData `json:"d"`
}

func getTxHashStr(txData []byte) string {
	txHash := sha256.Sum256(txData)
	return mustEncodeBase64URL(txHash[:])
}
