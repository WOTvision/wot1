package main

// A transaction output
type TxOutput struct {
	PubKey string `json:"k"`
	Amount uint64 `json:"a"`
	Data   string `json:"d"` // optional
}

// Tx is a transaction. It's usually kept as a JSON-serialized string in a block
type Tx struct {
	Version        uint              `json:"v"`
	SigningPubKey  string            `json:"k"`
	Flags          []string          `json:"f"` // "coinbase"
	Outputs        []TxOutput        `json:"o"`
	MinerFeeAmount uint              `json:"m"`
	Data           map[string]string `json:"d"`
}
