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
		id				INTEGER PRIMARY KEY,
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
	"document": `
	CREATE TABLE IF NOT EXISTS document (
		publisher_id    INTEGER PRIMARY KEY REFERENCES publisher(id),
		id				VARCHAR NOT NULL,
		block			INTEGER NOT NULL REFERENCES block(id)
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
	"fact_publisher_idx":      `CREATE UNIQUE INDEX IF NOT EXISTS fact_publisher_idx ON fact(publisher_id, key)`,
	"document_idx":            `CREATE UNIQUE INDEX IF NOT EXISTS document_idx ON document(publisher_id, id)`,
	"utxo_idx":                `CREATE INDEX IF NOT EXISTS utxo_idx ON utxo(txhash, n)`,
}

type Publisher struct {
	ID              int
	Name            string
	CurrentPubKey   string
	CurrentPubKeyID int
	SinceBlock      int
	ToBlock         int
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
	_, err = db.Exec("PRAGMA foreign_keys")
	if err != nil {
		log.Fatal(err)
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
		// Verify tx signature
		tx, err := btx.VerifyBasics()
		if err != nil {
			dbtx.Rollback()
			return err
		}

		// Check inputs and outputs

		// Import tx payload document data
		publisher, err := dbGetPublisherbyKey(tx.Data["_key"], height)
		if err != nil {
			if height > 0 {
				dbtx.Rollback()
				return fmt.Errorf("Publisher not found for key %s: %s", tx.Data["_key"], err.Error())
			}
			// Special handling for the genesis block
			publisher, err = dbIntroducePublisher(dbtx, &btx, &tx, 0)
			if err != nil {
				dbtx.Rollback()
				return err
			}
		}

		for key, value := range tx.Data {
			if key[0] == '_' {
				continue
			}
			_, err := db.Exec("INSERT OR REPLACE INTO fact(publisher_id, key, value) VALUES (?, ?, ?)", publisher.ID, key, value)
			if err != nil {
				dbtx.Rollback()
				return err
			}
		}
		err = dbSaveDocument(publisher, &tx, height)
		if err != nil {
			dbtx.Rollback()
			return err
		}
	}

	err = dbtx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func dbGetPublisherbyKey(pubkey string, atBlock int) (*Publisher, error) {
	p := Publisher{}
	err := db.QueryRow("SELECT id, publisher_id, since_block, to_block FROM publisher_pubkey WHERE pubkey=? ORDER BY since_block DESC", pubkey).Scan(&p.CurrentPubKeyID, &p.ID, &p.SinceBlock, &p.ToBlock)
	if err != nil {
		return nil, fmt.Errorf("Error getting publisher for pubkey %s", pubkey)
	}
	err = db.QueryRow("SELECT name FROM publisher WHERE id=?", p.ID).Scan(&p.Name)
	if err != nil {
		return nil, fmt.Errorf("Error getting publisher by ID %d", p.ID)
	}
	if p.SinceBlock >= atBlock && (p.ToBlock > 0 && atBlock <= p.ToBlock) {
		return &p, nil
	}
	return nil, fmt.Errorf("Publisher's key has expired")
}

func dbSaveDocument(publisher *Publisher, tx *Tx, height int) error {
	_, err := db.Exec("INSERT OR REPLACE INTO document (publisher_id, id, block) VALUES (?, ?, ?)", publisher.ID, tx.Data["_id"], height)
	return err
}

func dbIntroducePublisher(dbtx *sql.Tx, btx *BlockTransaction, tx *Tx, height int) (*Publisher, error) {
	id, ok := tx.Data["_id"]
	if !ok {
		return nil, fmt.Errorf("Missing _id in %s", btx.TxHash)
	}
	if id != "_intro" {
		return nil, fmt.Errorf("Trying to introduce a publisher without _id in %s", btx.TxHash)
	}

	name, ok := tx.Data["_name"]
	if !ok {
		return nil, fmt.Errorf("Trying to introduce a publisher without _name in %s", btx.TxHash)
	}

	pubKey, ok := tx.Data["_key"]
	if !ok {
		return nil, fmt.Errorf("Trying to introduce a publisher without _key in %s", btx.TxHash)
	}

	publisherID := 0
	publisherPubKeyID := 0
	publisherSinceBlock := 0
	err := db.QueryRow("SELECT publisher_id, id, since_block FROM publisher_pubkey WHERE pubkey=?", pubKey).Scan(&publisherID, &publisherPubKeyID, &publisherSinceBlock)
	if err == nil {
		// The publisher already exists, so this operation can only replace its key
		newKey, ok := tx.Data["_newkey"]
		if !ok {
			return nil, fmt.Errorf("Trying to re-introduce (replace key) a publisher without _newkey in %s", btx.TxHash)
		}
		res, err := dbtx.Exec("INSERT INTO publisher_pubkey (publisher_id, pubkey, since_block) VALUES (?, ?, ?)", publisherID, newKey, height)
		if err != nil {
			dbtx.Rollback()
			return nil, err
		}
		lastPubKeyID, err := res.LastInsertId()
		if err != nil {
			dbtx.Rollback()
			return nil, err
		}
		publisherPubKeyID = int(lastPubKeyID)
		_, err = dbtx.Exec("UPDATE publisher SET name=? WHERE id=?", name, publisherID)
		if err != nil {
			dbtx.Rollback()
			return nil, err
		}
	} else {
		// Brand new publisher
		res, err := dbtx.Exec("INSERT INTO publisher (name) VALUES (?)", name)
		if err != nil {
			return nil, err
		}
		lastID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		publisherID := int(lastID)
		res, err = dbtx.Exec("INSERT INTO publisher_pubkey (publisher_id, pubkey, since_block) VALUES (?,?,?)", publisherID, pubKey, height)
		if err != nil {
			return nil, err
		}
		lastKeyID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		publisherPubKeyID = int(lastKeyID)
		publisherSinceBlock = height
	}

	p := Publisher{ID: publisherID, CurrentPubKeyID: publisherPubKeyID, CurrentPubKey: pubKey, Name: name, SinceBlock: publisherSinceBlock}
	return &p, nil
}
