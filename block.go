package main

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

type BlockHeader struct {
	Hash string
}

type BlockTransaction struct {
	TxHash    string   `json:"h"`
	Flags     []string `json:"f"`
	TxData    string   `json:"t"`
	Signature string   `json:"s"`
}

type BlockWithHeader struct {
	BlockHeader
	Block
}

type Block struct {
	PreviousBlockHash string             `json:"p"`
	TimeUTC           uint               `json:"T"`
	Nonce             uint               `json:"n"`
	Flags             []string           `json:"f"`
	Transactions      []BlockTransaction `json:"t"`
}

var GenesisBlock = BlockWithHeader{
	BlockHeader: BlockHeader{
		Hash: "gvm_1rz4eoiq65mIm7vquolJbflQKbuTp3TQ38TKj0U",
	},
	Block: Block{
		PreviousBlockHash: "",
		TimeUTC:           1518733930,
		Nonce:             44990,
		Flags:             []string{"genesis"},
		Transactions: []BlockTransaction{
			BlockTransaction{
				TxHash:    "jghwrKFT-y9KYiLdk3eWCH-vB47nSxjMwEh0yhF3hM0",
				Flags:     []string{"coinbase"},
				TxData:    `{"v":1,"i":null,"o":[{"k":"WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY","a":100000}],"d":{"genesis":"The Guardian, 15th Feb 2018, \"Trump again emphasizes 'mental health' over gun control after Florida shooting\"","comment":"Peace among worlds!","_id":"_intro","_key":"WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY","_name":"WOTvision"}}`,
				Signature: "mev6DW0qRiNlflSMhHIKJ2D-sMM10vKW0Z8YNbnGwqdojoQnBBrYQHo1coAEQC2W2CroAv633NfegqLa6i-vAQ",
			},
		},
	},
}

const GenesisBlockDifficulty = 8 // number of zeroes

func (b *Block) Serialise(w io.Writer) error {
	jb, err := json.Marshal(b)
	if err != nil {
		return err
	}
	//log.Println(string(jb))
	n, err := w.Write(jb)
	if err != nil {
		return err
	}
	if n != len(jb) {
		return fmt.Errorf("Write error while serialising block: all data not written")
	}
	return nil
}

func (b *Block) Hash() []byte {
	h := sha512.New512_256()
	err := b.Serialise(h)
	if err != nil {
		log.Panicln("Cannot hash block:", err)
	}
	return h.Sum(nil)
}

// Adjusts nonce so that the hash of the block begins with "diff" zero bits
func (b *Block) Mine(diff int) {
	for {
		h := b.Hash()
		if countStartZeroBits(h) == diff {
			return
		}
		b.Nonce++
		if b.Nonce%10000 == 0 {
			fmt.Println(b.Nonce)
		}
	}
}

func initGenesis() {
	for _, btx := range GenesisBlock.Transactions {
		tx := Tx{}
		err := json.Unmarshal([]byte(btx.TxData), &tx)
		if err != nil {
			log.Panic(err)
		}
		txHash := sha256.Sum256([]byte(btx.TxData))
		if mustEncodeBase64URL(txHash[:]) != btx.TxHash {
			log.Fatalln("Unexpected genesis block tx hash. Expecting", mustEncodeBase64URL(txHash[:]), "got", btx.TxHash)
		}
		k, err := DecodePublicKeyString(tx.Data["_key"])
		if err != nil {
			log.Fatal(err)
		}
		err = k.VerifyRaw([]byte(btx.TxData), mustDecodeBase64URL(btx.Signature))
		if err != nil {
			log.Fatalln("Error verifying genesis block tx:", err)
		}
	}

	/*
		GenesisBlock.Mine(GenesisBlockDifficulty)
		log.Println(GenesisBlock.Nonce)
	*/

	bHash := GenesisBlock.Block.Hash()
	if mustEncodeBase64URL(bHash) != GenesisBlock.BlockHeader.Hash {
		log.Fatalln("Genesis block has failed hash check. Expecting", mustEncodeBase64URL(bHash), "got", GenesisBlock.BlockHeader.Hash)
	}
	if countStartZeroBits(bHash) != GenesisBlockDifficulty {
		log.Fatalln("Genesis block difficulty mismatch.")
	}
	log.Println("Genesis block ok.")
}
