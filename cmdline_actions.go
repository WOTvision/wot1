package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"time"
)

var reSqBrackets = regexp.MustCompile(`\[.+]\]`)

func showHelp() {
	fmt.Println("Usage:", os.Args[0], "<command> [argument...]")
	fmt.Println("Available commands:")
	fmt.Println("\thelp\t\tShows this help message.")
	fmt.Println("\tcreatewallet\tCreates a new wallet file. Expected arguments: filename wallet_name password.")
	fmt.Println("\tcreatekey\tCreates a new key in the current wallet (", path.Join(*dataDir, *walletFileName), "). Expected arguments: key_name password.")
	fmt.Println("\t\t\tNote: key_name is the publisher name when the key gets introduced in the blockchain.")
	fmt.Println("\tsignjson\tSigns a JSON document string with the specified key. Expected arguments: key_name password json.")
	fmt.Println("\tlistkeys\tLists the keys in the current wallet.")
	fmt.Println()
	fmt.Println("If started without a command specified, a blockchain node will be started.")
}

func processSimpleCmdLineActions() bool {
	cmd := flag.Arg(0)
	if cmd == "help" {
		showHelp()
		os.Exit(0)
	} else if cmd == "createwallet" {
		if flag.NArg() != 4 {
			fmt.Println("Expecting arguments: filename wallet_name password")
			os.Exit(1)
		}
		filename := flag.Arg(1)
		name := flag.Arg(2)
		password := flag.Arg(3)

		w := Wallet{Name: name, Flags: []string{WalletFlagAES256Keys}}
		err := w.createKey("default", password)
		if err != nil {
			log.Fatalln("Cannot create key:", err)
		}

		err = w.Save(filename)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fmt.Sprintf("Created a key named '%s'", w.Keys[0].Name))
		return true
	} else if cmd == "createkey" {
		if flag.NArg() != 3 {
			fmt.Println("Expecting arguments: key_name password")
			os.Exit(1)
		}
		keyName := flag.Arg(1)
		keyPassword := flag.Arg(2)

		initWallet(false)

		err := currentWallet.createKey(keyName, keyPassword)
		if err != nil {
			log.Fatalln("Cannot create key:", err)
		}
		err = currentWallet.Save(currentWalletFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fmt.Sprintf("Created a key named '%s'", currentWallet.Keys[len(currentWallet.Keys)-1].Name))
		return true
	} else if cmd == "signjson" {
		if flag.NArg() != 4 {
			fmt.Println("Expecting arguments: key_name password json")
			os.Exit(1)
		}
		keyName := flag.Arg(1)
		keyPassword := flag.Arg(2)
		jsonToSign := []byte(flag.Arg(3))

		initWallet(false)

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
	} else if cmd == "listkeys" {
		initWallet(false)
		if len(currentWallet.Keys) < 1 {
			log.Fatal("No keys in current wallet")
		}
		for _, key := range currentWallet.Keys {
			fmt.Println(fmt.Sprintf("%-25s %v %v", key.Name, key.CreationTime.Format(time.RFC3339), key.Flags))
		}
		return true
	}
	return false
}

func processCmdLineActions() bool {
	cmd := flag.Arg(0)
	if cmd == "createwallet" {
		return true
	} else if cmd == "signjson" {
		return true
	}
	return false
}
