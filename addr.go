package ble

import (
	"encoding/hex"
	"strings"
)

// Addr represents a network end point address.
// It's MAC address on Linux or Device UUID on OS X.
type Addr interface {
	String() string
	Bytes() []byte
}

// NewAddr creates an Addr from string
func NewAddr(s string) Addr {
	return addr(strings.ToLower(s))
}

type addr string

func (a addr) String() string {
	return string(a)
}

func (a addr) Bytes() []byte {
	hexStr := strings.Replace(a.String(), ":", "", -1)

	out, _ := hex.DecodeString(hexStr)

	return out
}
