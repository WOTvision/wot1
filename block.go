package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
)

type BlockHeader struct {
	Hash              []byte
	PreviousBlockHash []byte
	TimeUTC           int
}

const BlockTransactionFlagUncompressed = 1

type BlockTransaction struct {
	TxHash    []byte
	Flags     int
	TxData    []byte
	Signature []byte
}

type Block struct {
	Header       BlockHeader
	Transactions []BlockTransaction
}

var GenesisBlock = Block{
	Header: BlockHeader{
		Hash:              []byte{},
		PreviousBlockHash: []byte{},
		TimeUTC:           1518733930,
	},
	Transactions: []BlockTransaction{
		BlockTransaction{
			TxHash:    mustDecodeBase64URL("8O7mUP5NWPw3ABs0dlDjKn-lizR4eziBDVb6FE8eaEs"),
			Flags:     BlockTransactionFlagUncompressed,
			TxData:    []byte(`{"v":1,"t":0,"i":null,"o":[{"k":"WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY","a":1000}],"d":{"genesis":"The Guardian, 15th Feb 2018, \"Trump again emphasizes 'mental health' over gun control after Florida shooting\"","comment":"Peace among worlds!","_introkey":"WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY"}}`),
			Signature: mustDecodeBase64URL("KBnOVGSJtDne-AO0qrco_FRUnj86DKZ_MBqJpJ-Q2cbRdVIZX0IulmqH0Q2g2la-p1wl-rGwFrU27XvUtmolDA"),
		},
	},
}

func (bt *BlockTransaction) Serialise(w io.Writer) {
	binary.Write(w, binary.LittleEndian, bt)
}

func (b *Block) Serialise(w io.Writer) {
	for _, tx := range b.Transactions {
		tx.Serialise(w)
	}
}

func (b *Block) Hash() []byte {
	h := sha256.New()
	b.Serialise(h)
	return h.Sum(nil)
}

func initGenesis() {
	for _, btx := range GenesisBlock.Transactions {
		tx := Tx{}
		err := json.Unmarshal(btx.TxData, &tx)
		if err != nil {
			log.Panic(err)
		}
		txHash := sha256.Sum256(btx.TxData)
		if mustEncodeBase64URL(txHash[:]) != mustEncodeBase64URL(btx.TxHash) {
			log.Fatalln("Unexpected genesis block tx hash. Expecting", mustEncodeBase64URL(txHash[:]), "got", mustEncodeBase64URL(btx.TxHash))
		}
		k, err := DecodePublicKeyString(tx.Data["_introkey"])
		if err != nil {
			log.Fatal(err)
		}
		err = k.VerifyRaw(btx.TxData, btx.Signature)
		if err != nil {
			log.Fatalln("Error verifying genesis block tx:", err)
		}
	}
	log.Println("Genesis block ok.")
}
