package main

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
)

const blocksDirectoryName = "blocks"
const blockFileFormat = "%10d %s.gz"
const blockFileGlob = "*.gz"

var blocksDir = ""
var currentBlockHeight = 0

var reBlockFilename = regexp.MustCompile(`^([0-9]+) (.+?)\.gz$`)

func initDataDir() {
	log.Println("Data directory:", *dataDir)
	blocksDir = path.Join(*dataDir, blocksDirectoryName)
	if _, err := os.Stat(*dataDir); os.IsNotExist(err) {
		bootstrapDataDir()
	}
}

func getBlockFilename(height int, b BlockWithHeader) string {
	return path.Join(blocksDir, fmt.Sprintf(blockFileFormat, height, b.BlockHeader.Hash))
}

func getBlockDataFromFilename(fn string) (int, string) {
	base := filepath.Base(fn)
	m := reBlockFilename.FindStringSubmatch(base)
	height, err := strconv.Atoi(m[1])
	if err != nil {
		return -1, ""
	}
	return height, m[2]
}

func bootstrapDataDir() {
	err := os.Mkdir(*dataDir, 0750)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	err = os.Mkdir(blocksDir, 0750)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	currentBlockHeight = 0
	// Write the genesis block data file. All block files are
	f, err := os.Create(getBlockFilename(currentBlockHeight, GenesisBlock))
	if err != nil {
		log.Fatal(err)
	}
	zf, err := gzip.NewWriterLevel(f, 9)
	if err != nil {
		log.Fatal(err)
	}
	err = GenesisBlock.Block.Serialise(zf)
	if err != nil {
		log.Fatal(err)
	}
	err = zf.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}
