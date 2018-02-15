package main

// TxInput represents transaction input
type TxInput struct {
	Hash  string `json:"h"`
	Index int    `json:"i"`
}

// A transaction output
type TxOutput struct {
	PubKey string `json:"k"`
	Amount int    `json:"a"`
}

// Tx is a transaction. It's usually kept as a JSON-serialized string in a block
type Tx struct {
	Version int               `json:"v"`
	Type    int               `json:"t"`
	Inputs  []TxInput         `json:"i"`
	Outputs []TxOutput        `json:"o"`
	Data    map[string]string `json:"d"`
}
