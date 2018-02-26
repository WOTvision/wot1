package main

import (
	"compress/gzip"
	"crypto/sha512"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

const sqliteDatabaseName = "wot.sqlite3"

var dbTables = map[string]string{
	"block": `
	CREATE TABLE IF NOT EXISTS block (
		height     		INTEGER PRIMARY KEY,
		hash       		TEXT NOT NULL UNIQUE,
		ts		 		INTEGER
	)`,
	"publisher": `
	CREATE TABLE IF NOT EXISTS publisher (
		id				INTEGER PRIMARY KEY,
		name			TEXT NOT NULL,
		data			TEXT
	)`,
	"publisher_pubkey": `
	CREATE TABLE IF NOT EXISTS publisher_pubkey (
		publisher_id 	INTEGER NOT NULL REFERENCES publisher(id),
		pubkey			TEXT NOT NULL,
		since_block		INTEGER NOT NULL REFERENCES block(height),
		to_block		INTEGER REFERENCES block(height)
	)`,
	"fact": `
	CREATE TABLE IF NOT EXISTS fact (
		publisher_id 	INTEGER NOT NULL REFERENCES publisher(id),
		key				TEXT NOT NULL,
		value			TEXT NOT NULL
	)`,
	"utxo": `
	CREATE TABLE IF NOT EXISTS utxo (
		txhash			TEXT NOT NULL,
		n				INTEGER NOT NULL,
		amount			INTEGER NOT NULL
	)`,
}

var dbTableIndexes = map[string]string{
	"publisher_pubkey_idx":    `CREATE INDEX IF NOT EXISTS publisher_pubkey_idx ON publisher_pubkey(pubkey)`,
	"publisher_pubkey_id_idx": `CREATE INDEX IF NOT EXISTS publisher_pubkey_id_idx ON publisher_pubkey(publisher_id)`,
	"fact_publisher_idx":      `CREATE INDEX IF NOT EXISTS fact_publisher_idx ON fact(publisher_id, key)`,
	"utxo_idx":                `CREATE INDEX IF NOT EXISTS utxo_idx ON utxo(txhash, n)`,
}

var db *sql.DB

func dbFilePresent() bool {
	dbName := path.Join(*dataDir, sqliteDatabaseName)
	_, err := os.Stat(dbName)
	return err == nil
}

func initDatabase() {
	var err error
	exists := true
	dbName := path.Join(*dataDir, sqliteDatabaseName)
	if _, err = os.Stat(dbName); err != nil {
		exists = false
	}
	db, err = sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		_, err = db.Exec("PRAGMA journal_mode=WAL")
		if err != nil {
			log.Fatal(err)
		}
	}
	for tname, tdef := range dbTables {
		_, err := db.Exec(tdef)
		if err != nil {
			log.Println("Error creating table", tname)
			log.Fatal(err)
		}
	}
	for iname, idef := range dbTableIndexes {
		_, err := db.Exec(idef)
		if err != nil {
			log.Println("Error creating index", iname)
			log.Fatal(err)
		}
	}
	blocks, err := filepath.Glob(path.Join(blocksDir, blockFileGlob))
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(blocks)
	for _, bFile := range blocks {
		height, hash := getBlockDataFromFilename(bFile)
		if !dbExistsBlockByHeight(height) {
			err := dbImportBlockFile(bFile, height, hash)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func shutdownDatabase() {
	db.Exec("PRAGMA optimize")
	db.Close()
}

func dbExistsBlockByHeight(height int) bool {
	count := 0
	err := db.QueryRow("SELECT COUNT(*) FROM block WHERE height=?", height).Scan(&count)
	if err != nil {
		log.Panic(err)
	}
	return count != 0
}

func dbImportBlockFile(fn string, height int, hash string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	zf, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer zf.Close()
	bData, err := ioutil.ReadAll(zf)
	if err != nil {
		return err
	}
	return dbImportBlock(bData, height, hash)
}

func dbImportBlock(bData []byte, height int, hash string) error {
	log.Println("Importing block", hash, "at", height)

	// Check if hash matches
	h := sha512.New512_256()
	h.Write(bData)
	bHash := h.Sum(nil)
	if mustEncodeBase64URL(bHash) != hash {
		return fmt.Errorf("Block hash doesn't match block data: Expecting %s got %s", mustEncodeBase64URL(bHash), hash)
	}

	b := BlockWithHeader{BlockHeader: BlockHeader{Hash: hash}}
	err := json.Unmarshal(bData, &b.Block)
	if err != nil {
		return fmt.Errorf("Cannot unmarshall block: %s", err.Error())
	}

	dbtx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, btx := range b.Transactions {
		// verify tx signature

		// import tx data
		tx := Tx{}
		err := json.Unmarshal([]byte(btx.TxData), &tx)
		if err != nil {
			return fmt.Errorf("Cannot unmarshall tx: %s", btx.TxHash)
		}
	}

	err = dbtx.Commit()
	if err != nil {
		return err
	}

	return nil
}
