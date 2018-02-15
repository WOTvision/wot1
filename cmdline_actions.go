package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
)

var reSqBrackets = regexp.MustCompile(`\[.+]\]`)

func processCmdLineActions() bool {
	cmd := flag.Arg(0)
	if cmd == "createwallet" {
		if flag.NArg() != 4 {
			fmt.Println("Expecting arguments: filename name password")
			os.Exit(1)
		}
		filename := flag.Arg(1)
		name := flag.Arg(2)
		password := flag.Arg(3)

		w := Wallet{Name: name, Flags: []string{WalletFlagAES256Keys}}
		err := w.createKey("default", password)
		if err != nil {
			log.Fatal("Cannot create key:", err)
		}

		err = w.Save(filename)
		if err != nil {
			log.Fatal(err)
		}

		return true
	}
	return false
}
