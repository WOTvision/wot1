package main

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
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
	StateHash         string             `json:"s"`
}

var GenesisBlock = BlockWithHeader{
	BlockHeader: BlockHeader{
		Hash: "iU3HyYPlr-nxwGSm_lFw2ERQ7fibxqmiRIaGc9GV-NE",
	},
	Block: Block{
		PreviousBlockHash: "",
		TimeUTC:           1518733930,
		Nonce:             106367,
		Flags:             []string{"genesis"},
		Transactions: []BlockTransaction{
			BlockTransaction{
				TxHash:    "33eH1RtE7NmDkkE8hinXBcptWw9zUljt8nv3UtM7bIc",
				Flags:     []string{},
				TxData:    `{"v":1,"f":["coinbase"],"k":"WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY","o":[{"k":"WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY","a":1000000,"n":1}],"d":{"genesis":"The Guardian, 15th Feb 2018, \"Trump again emphasizes 'mental health' over gun control after Florida shooting\"","comment":"Peace among worlds!","_id":"_intro","_key":"WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY","_name":"WOTvision"}}`,
				Signature: "PMCJiphBeNR6nggA7NQcIzHZ31jqQa80gJVV70qLigiM594tVBAvSXFDxGSqgL-H_K1wcnptTJ16eGvqXFffAw",
			},
		},
		StateHash: "nof_udomIBf8I2-PavfShu8q7ii2WwFROy1ZaxRvxow",
	},
}

const GenesisBlockDifficulty = 8 // number of zero bits

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

func (btx *BlockTransaction) VerifyBasics() (Tx, error) {
	txDataBytes := []byte(btx.TxData)

	tx := Tx{}
	txHash := sha256.Sum256(txDataBytes)
	if mustEncodeBase64URL(txHash[:]) != btx.TxHash {
		return tx, fmt.Errorf("Invalid tx hash: %s", btx.TxHash)
	}
	sig, err := base64.RawURLEncoding.DecodeString(btx.Signature)
	if err != nil {
		return tx, err
	}
	err = json.Unmarshal(txDataBytes, &tx)
	if err != nil {
		return tx, fmt.Errorf("Cannot unmarshall tx: %s", btx.TxHash)
	}
	_, ok := tx.Data["_id"]
	if !ok {
		return tx, fmt.Errorf("Missing _id in tx data: %s", btx.TxHash)
	}
	k, err := DecodePublicKeyString(tx.SigningPubKey)
	if err != nil {
		return tx, err
	}
	if err = k.VerifyRaw(txDataBytes, sig); err != nil {
		return tx, fmt.Errorf("Signature doesn't match _key: %s: %s", btx.TxHash, err.Error())
	}
	return tx, nil
}

func initGenesis() {
	states := AccountStates{}
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
			log.Fatalln("Error verifying genesis block tx signature:", err)
		}
		for _, o := range tx.Outputs {
			if states[o.PubKey] == nil {
				states[o.PubKey] = &RawAccountState{Balance: o.Amount, Nonce: o.Nonce}
			} else {
				states[o.PubKey].Balance += o.Amount
				if o.Nonce != states[o.PubKey].Nonce+1 {
					log.Fatalln("nonce out of sync in genesis block")
				}
				states[o.PubKey].Nonce = o.Nonce
			}
		}
	}
	strStateHash := states.getStrHash()
	if GenesisBlock.StateHash != strStateHash {
		log.Println("New genesis state:", jsonifyWhatever(states))
		log.Fatalln("Unexpected genesis state hash. Expecting", strStateHash, "got", GenesisBlock.StateHash)
	}

	bHash := GenesisBlock.Block.Hash()
	if mustEncodeBase64URL(bHash) != GenesisBlock.BlockHeader.Hash {
		log.Fatalln("Genesis block has failed hash check. Expecting", mustEncodeBase64URL(bHash), "got", GenesisBlock.BlockHeader.Hash)
	}
	if countStartZeroBits(bHash) != GenesisBlockDifficulty {
		GenesisBlock.Mine(GenesisBlockDifficulty)
		log.Println("Mined nonce:", GenesisBlock.Nonce)
		log.Fatalln("Genesis block difficulty mismatch.")
	}
	log.Println("Genesis block ok.")
}

func getCoinbaseAtHeight(height int) uint64 {
	return 100 * OneCoin
}
