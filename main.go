package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"os/user"
	"syscall"
	"time"
)

const (
	eventQuit = iota
)

type sysEventMessage struct {
	event int
	idata int
}

var sysEventChannel = make(chan sysEventMessage, 5)
var startTime = time.Now()

var logFileName = flag.String("log", "/tmp/wot1.log", "Log file ('-' for only stderr)")
var walletFileName = flag.String("wallet", DefaultWalletFilename, "Wallet filename")
var dataDir = flag.String("datadir", "~/.wot", "Data directory for the blockchain")

func main() {
	flag.Parse()

	if *logFileName != "-" {
		f, err := os.OpenFile(*logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
		if err != nil {
			log.Panic("Cannot open log file " + *logFileName)
		}
		defer f.Close()
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	} else {
		log.SetOutput(os.Stderr)
	}

	if (*dataDir)[0] == '~' {
		usr, err := user.Current()
		if err != nil {
			log.Fatal("Cannot get current user's home directory")
		}
		*dataDir = usr.HomeDir + (*dataDir)[1:]
	}

	if len(flag.Args()) > 0 {
		if processSimpleCmdLineActions() {
			return
		}
	}

	initGenesis()
	initDataDir()
	initDatabase()
	initWallet()

	if len(flag.Args()) > 0 {
		if processCmdLineActions() {
			return
		}
	}

	log.Println("Starting up...")
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT)

	go webServer()
	for {
		select {
		case msg := <-sysEventChannel:
			switch msg.event {
			case eventQuit:
				log.Println("Exiting")
				os.Exit(msg.idata)
			}
		case sig := <-sigChannel:
			switch sig {
			case syscall.SIGINT:
				sysEventChannel <- sysEventMessage{event: eventQuit, idata: 0}
				log.Println("^C detected")
			}
		case <-time.After(60 * time.Second):
			log.Println("Tick.")
		}
	}
}
