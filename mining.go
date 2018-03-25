package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

var currentDifficulty = GenesisBlockDifficulty

func miningRig() {
	log.Println("Starting the PoW miner...")
	if *miningRewardAddress == "" {
		if len(currentWallet.Keys) == 0 {
			log.Println("No miningAddress specified and no keys in current wallet. Stopping the miner.")
			return
		}
		*miningRewardAddress = currentWallet.Keys[0].Public
	}
	for {
		var min, max sql.NullInt64
		dbtx, err := db.Begin()
		if err != nil {
			log.Println(err)
			continue
		}
		err = dbtx.QueryRow("SELECT MIN(id), MAX(id) FROM utx").Scan(&min, &max)
		if err != nil {
			log.Println("Mining block min/max error", err)
			continue
		}
		if min.Valid && max.Valid {
			lastHash := ""
			lastHeight := 0
			err = dbtx.QueryRow("SELECT hash, height FROM block ORDER BY height DESC LIMIT 1").Scan(&lastHash, &lastHeight)
			if err != nil {
				log.Println(err)
				continue
			}
			err = mineBlock(dbtx, uint64(min.Int64), uint64(max.Int64), lastHeight, lastHash, *miningRewardAddress)
			if err == nil {
				_, err = dbtx.Exec("DELETE FROM utx WHERE id BETWEEN ? AND ?", min.Int64, max.Int64)
				if err != nil {
					log.Panic(err)
				}
				dbtx.Commit()
			} else {
				log.Println(err)
				dbtx.Rollback()
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func mineBlock(dbtx *sql.Tx, min, max uint64, prevHeight int, prevHash string, rewardAddress string) error {
	newHeight := prevHeight + 1
	block := BlockWithHeader{Block: Block{TimeUTC: time.Now().Unix(), PreviousBlockHash: prevHash, Transactions: []BlockTransaction{}}}
	coinbaseReward := getCoinbaseAtHeight(newHeight)
	var err error
	for i := min; i <= max; i++ {
		txData := ""
		err = dbtx.QueryRow("SELECT tx FROM utx WHERE id=?", i).Scan(&txData)
		if err != nil {
			return err
		}
		btx := BlockTransaction{}
		err = json.Unmarshal([]byte(txData), &btx)
		if err != nil {
			return err
		}
		tx, err := btx.VerifyBasics()
		if err != nil {
			return err
		}
		coinbaseReward += tx.MinerFeeAmount
		block.Transactions = append(block.Transactions, btx)
	}
	coinbaseTx := Tx{Flags: []string{"coinbase"}, Version: CurrentTxVersion, Outputs: []TxOutput{TxOutput{PubKey: rewardAddress, Amount: coinbaseReward}}}
	coinbaseTxData := jsonifyWhateverToBytes(coinbaseTx)
	txHash := getTxHashStr(coinbaseTxData)
	coinbaseBtx := BlockTransaction{TxHash: txHash, TxData: string(coinbaseTxData)}
	block.Transactions = append([]BlockTransaction{coinbaseBtx}, block.Transactions...)
	block.StateHash, err = dbSimStateHashStr(dbtx, block)
	if err != nil {
		return err
	}

	block.BlockHeader.Hash = block.Mine(currentDifficulty)

	err = dataDirSaveBlock(block, newHeight)
	if err != nil {
		log.Fatal(err)
	}
	err = dbImportCheckedBlock(dbtx, block, newHeight, block.BlockHeader.Hash)
	if err != nil {
		dataDirDeleteBlock(block, newHeight)
		return err
	}
	log.Printf("!! Mined block %s at %d, %d transaction(s)\n", block.BlockHeader.Hash, newHeight, max-min+1)
	return nil
}
