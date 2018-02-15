package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
)

var reSqBrackets = regexp.MustCompile(`\[.+]\]`)

func processCmdLineActions() bool {
	cmd := flag.Arg(0)
	if cmd == "createuser" {
		if flag.NArg() != 3 {
			fmt.Println("Expecting arguments: email password")
			os.Exit(1)
		}
		return true
	}
	return false
}
