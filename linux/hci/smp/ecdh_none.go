// +build !blesmp

package smp

import (
	"crypto"
	"errors"
)

type ECDHKeys struct {
	public  crypto.PublicKey
	private crypto.PrivateKey
}

func GenerateKeys() (*ECDHKeys, error) {
	return nil, errors.New("not implemented")
}

func UnmarshalPublicKey(b []byte) (crypto.PublicKey, bool) {
	return nil, false
}

func MarshalPublicKeyXY(k crypto.PublicKey) []byte {
	return nil
}

func MarshalPublicKeyX(k crypto.PublicKey) []byte {
	return nil
}

func GenerateSecret(prv crypto.PrivateKey, pub crypto.PublicKey) ([]byte, error) {
	return nil, errors.New("not implemented")
}
