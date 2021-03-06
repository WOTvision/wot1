package main

//#include <string.h>
import "C"

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"
)

func int16atobytes(a []int16) []byte {
	b := make([]byte, len(a)*2)

	pa := unsafe.Pointer(&a[0])
	pb := unsafe.Pointer(&b[0])

	C.memcpy(pb, pa, C.size_t(len(b)))

	return b
}

func bytestoint16a(b []byte) []int16 {
	a := make([]int16, len(b)/2)

	pa := unsafe.Pointer(&a[0])
	pb := unsafe.Pointer(&b[0])

	C.memcpy(pa, pb, C.size_t(len(b)))
	return a
}

func inIntSlice(i int, si []int) bool {
	for _, j := range si {
		if i == j {
			return true
		}
	}
	return false
}

func jsonifyWhatever(i interface{}) string {
	jsonb, err := json.Marshal(i)
	if err != nil {
		log.Panic(err)
	}
	return string(jsonb)
}

// Converts whatever to a json string in bytes
func jsonifyWhateverToBytes(i interface{}) []byte {
	jsonb, err := json.Marshal(i)
	if err != nil {
		log.Panic(err)
	}
	return jsonb
}

// WithMutex extends the Mutex type with the convenient .With(func) function
type WithMutex struct {
	sync.Mutex
}

// With executes the given function with the mutex locked
func (m *WithMutex) With(f func()) {
	m.Mutex.Lock()
	f()
	m.Mutex.Unlock()
}

// Converts the given Unix timestamp to time.Time
func unixTimeStampToUTCTime(ts int) time.Time {
	return time.Unix(int64(ts), 0)
}

// Gets the current Unix timestamp in UTC
func getNowUTC() int64 {
	return time.Now().UTC().Unix()
}

// Mashals the given map of strings to JSON
func stringMap2JsonBytes(m map[string]string) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		log.Panicln("Cannot json-ise the map:", err)
	}
	return b
}

// Returns a hex-encoded hash of the given byte slice
func hashBytesToHexString(b []byte) string {
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

// Returns a hex-encoded hash of the given file
func hashFileToHexString(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Checks if a string appears in the slice of strings
func inStringSlice(s string, a []string) bool {
	for _, ss := range a {
		if ss == s {
			return true
		}
	}
	return false
}

func bytesToBytesLiteral(buf []byte) string {
	parts := []string{}
	for _, b := range buf {
		parts = append(parts, fmt.Sprintf("0x%x", b))
	}
	return strings.Join(parts, ",")
}

func mustDecodeBase64URL(s string) []byte {
	buf, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		log.Panic(err)
	}
	return buf
}

func mustEncodeBase64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func countStartZeroBits(b []byte) int {
	nBits := 0
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			nBits += 8
		} else {
			for z := uint(7); z >= 0; z-- {
				if b[i]&(1<<z) == 0 {
					nBits++
				} else {
					break
				}
			}
		}
	}
	return nBits
}

func stringMapHasKey(m *map[string]string, k string) bool {
	_, ok := (*m)[k]
	return ok
}
