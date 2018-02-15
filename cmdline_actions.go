package main

import (
	"encoding/base64"
	"encoding/json"
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
		fmt.Println(fmt.Sprintf("Created a key named '%s'", w.Keys[0].Name))

		return true
	} else if cmd == "signjson" {
		if flag.NArg() != 4 {
			fmt.Println("Expecting arguments: key_name password json")
			os.Exit(1)
		}
		keyName := flag.Arg(1)
		keyPassword := flag.Arg(2)
		jsonToSign := []byte(flag.Arg(3))

		var ii interface{}
		err := json.Unmarshal(jsonToSign, &ii)
		if err != nil {
			log.Fatal(err)
		}

		if len(currentWallet.Keys) < 1 {
			log.Fatal("No keys in current wallet")
		}

		found := false
		for _, key := range currentWallet.Keys {
			if key.Name == keyName {
				found = true
				err := key.UnlockPrivateKey(keyPassword)
				if err != nil {
					log.Fatal(err)
				}
				sig, err := key.SignRaw(jsonToSign)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(base64.RawURLEncoding.EncodeToString(sig))
				fmt.Println(bytesToBytesLiteral(sig))
				break
			}
		}
		if !found {
			log.Fatal("Cannot find key", keyName)
		}
		return true
	}
	return false
}
