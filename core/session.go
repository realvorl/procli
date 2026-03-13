package core

import (
	"math/rand"
	"time"
)

const sessionAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateSessionCode(length int) string {
	if length <= 0 {
		length = 6
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	out := make([]byte, length)
	for i := range out {
		out[i] = sessionAlphabet[r.Intn(len(sessionAlphabet))]
	}

	return string(out)
}
