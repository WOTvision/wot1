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
const blockFileFormat = "%010d %s.gz"
const blockFileGlob = "*.gz"

var blocksDir = ""
var currentBlockHeight = 0

var reBlockFilename = regexp.MustCompile(`^([0-9]+) (.+?)\.gz$`)

func initDataDir() {
	log.Println("Data directory:", *dataDir)
	blocksDir = path.Join(*dataDir, blocksDirectoryName)
	if _, err := os.Stat(*dataDir); os.IsNotExist(err) {
		bootstrapDataDir()
	} else if !dbFilePresent() || countDataDirBlocks() < 1 {
		bootstrapDataDir()
	}
}

func countDataDirBlocks() int {
	blocks, err := filepath.Glob(path.Join(blocksDir, blockFileGlob))
	if err != nil {
		log.Println(err)
		return -1
	}
	return len(blocks)
}

func getBlockFilename(b BlockWithHeader, height int) string {
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
	err = dataDirSaveBlock(GenesisBlock, 0)
	if err != nil {
		log.Fatal(err)
	}
}

func dataDirSaveBlock(b BlockWithHeader, height int) error {
	fname := getBlockFilename(b, height)
	// Write the genesis block data file. All block files are gzipped.
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	zf, err := gzip.NewWriterLevel(f, 9)
	if err != nil {
		os.Remove(fname)
		return err
	}
	err = b.Block.Serialise(zf)
	if err != nil {
		os.Remove(fname)
		return err
	}
	err = zf.Close()
	if err != nil {
		os.Remove(fname)
		return err
	}
	err = f.Close()
	if err != nil {
		os.Remove(fname)
		return err
	}
	return nil
}

func dataDirDeleteBlock(b BlockWithHeader, height int) error {
	return os.Remove(getBlockFilename(b, height))
}
