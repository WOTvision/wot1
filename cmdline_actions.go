package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
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
	fmt.Println("\tsignjson\tSigns a JSON document string with the specified key. Expected arguments: key_name password json_document.")
	fmt.Println("\tlistkeys\tLists the keys in the current wallet.")
	fmt.Println("\tsend\tSends coins in a transactions, with optional JSON document. Expected arguments: from_key password to_key amount [json_document].")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("* If started without a command specified, a blockchain node will be started.")
	fmt.Println("* The json_document argument (where applicable) is literally a JSON string.")
}

func processSimpleCmdLineActions() bool {
	// Actions which may be done when the blockchain is not yet functional (initialised)
	cmd := flag.Arg(0)
	if cmd == "createwallet" {
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
			fmt.Println(fmt.Sprintf("%-25s %s %v %v", key.Name, key.Public, key.CreationTime.Format(time.RFC3339), key.Flags))
		}
		return true
	}
	return false
}

func processCmdLineActions() bool {
	// Actions which require the blockchain to be functional
	cmd := flag.Arg(0)
	if cmd == "help" {
		showHelp()
		os.Exit(0)
	} else if cmd == "send" {
		if len(currentWallet.Keys) < 1 {
			log.Fatal("No keys in current wallet")
		}
		if flag.NArg() < 5 {
			fmt.Println("Expecting arguments: from_key password to_key amount [json_document]")
			os.Exit(1)
		}
		fromKeyStr := flag.Arg(1)
		fromKeyPassword := flag.Arg(2)
		toKeyStr := flag.Arg(3)
		amountStr := flag.Arg(4)
		jsonDoc := flag.Arg(5) // optional
		var fromKey *WalletKey
		for kid, key := range currentWallet.Keys {
			if key.Name == fromKeyStr {
				fromKeyStr = key.Public
			}
			if key.Name == toKeyStr {
				toKeyStr = key.Public
			}
			if key.Public == fromKeyStr {
				fromKey = &(currentWallet.Keys[kid])
			}
		}
		if fromKey == nil {
			fmt.Println("The from_key argument must be in the current wallet")
			os.Exit(1)
		}
		if fromKeyStr == toKeyStr {
			fmt.Println("Warning: sending a tx from and to the same address")
		}
		f, _, err := big.ParseFloat(amountStr, 10, CoinDecimals, big.ToPositiveInf)
		if err != nil {
			fmt.Println("Invalid number:", amountStr, err.Error())
			os.Exit(1)
		}
		if f.Sign() < 0 {
			fmt.Println("Cannot send negative amounts")
			os.Exit(1)
		}
		f.Mul(f, big.NewFloat(OneCoin))
		amountInt, _ := f.Uint64()
		var doc PublishedData
		if jsonDoc != "" && json.Unmarshal([]byte(jsonDoc), &doc) != nil {
			fmt.Println("Invalid JSON document:", jsonDoc)
		}
		newNonce := uint64(1)
		dbtx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		states, err := dbGetStates(dbtx, []string{toKeyStr})
		if err != nil && err != sql.ErrNoRows {
			log.Fatal(err)
		}
		dbtx.Rollback()
		if len(states) != 0 {
			newNonce = states[toKeyStr].Nonce + 1
		}
		tx := Tx{Data: doc, SigningPubKey: fromKeyStr, Version: CurrentTxVersion, Outputs: []TxOutput{TxOutput{PubKey: toKeyStr, Amount: amountInt, Nonce: newNonce}}}
		txJSONBytes := jsonifyWhateverToBytes(tx)
		err = fromKey.UnlockPrivateKey(fromKeyPassword)
		if err != nil {
			log.Fatal(err)
		}
		sig, err := fromKey.SignRaw(txJSONBytes)
		if err != nil {
			log.Fatal(err)
		}
		strSig := mustEncodeBase64URL(sig)
		btx := BlockTransaction{TxHash: getTxHashStr(txJSONBytes), TxData: string(txJSONBytes), Signature: strSig}
		fmt.Println(jsonifyWhatever(btx))
		return true
	}
	return false
}
