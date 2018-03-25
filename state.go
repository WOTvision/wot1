package main

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// RawAccountState stores an acccount's (address') state
type RawAccountState struct {
	Balance uint64 `json:"b"`
	Nonce   uint64 `json:"n"`
	Data    string `json:"d"`
}

func (s *RawAccountState) AddBalance(amount uint64) {
	// XXX: check for overflows
	s.Balance += amount
}

func (s *RawAccountState) IncNonce() {
	s.Nonce++
}

type AccountStates map[string]*RawAccountState

func (states AccountStates) getHash() []byte {
	keys := []string{}
	for k := range states {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	jsonList := []string{}
	for _, k := range keys {
		jsonList = append(jsonList, fmt.Sprintf(`{"%s":{"b":%v,"n":%d,"d":%s}}`, k, states[k].Balance, states[k].Nonce, jsonifyWhatever(states[k].Data)))
	}
	jsonString := fmt.Sprintf("{%s}", strings.Join(jsonList, ","))
	hash := sha256.Sum256([]byte(jsonString))
	return hash[:]
}

func (states AccountStates) getStrHash() string {
	return mustEncodeBase64URL(states.getHash())
}
