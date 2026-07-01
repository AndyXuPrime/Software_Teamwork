package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
)

var requestIDCounter atomic.Uint64

func newRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "mcp_" + hex.EncodeToString([]byte{byte(requestIDCounter.Add(1))})
	}
	return "mcp_" + hex.EncodeToString(buf[:])
}
