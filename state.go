package main

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

func calcBalancesHash(balances map[string]uint64) []byte {
	keys := []string{}
	for k := range balances {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	balancesStrings := []string{}
	for _, k := range keys {
		balancesStrings = append(balancesStrings, fmt.Sprintf(`{"%s":%v}`, k, balances[k]))
	}
	jsonString := fmt.Sprintf("{%s}", strings.Join(balancesStrings, ","))
	hash := sha256.Sum256([]byte(jsonString))
	return hash[:]
}
