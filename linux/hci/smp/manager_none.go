// +build !blesmp

package smp

import (
	"errors"
	"time"

	"github.com/photostorm/ble"
	"github.com/photostorm/ble/linux/hci"
)

type PairingState int

const (
	Init PairingState = iota
	WaitPairingResponse
	WaitPublicKey
	WaitConfirm
	WaitRandom
	WaitDhKeyCheck
	Finished
	Error
)

type manager struct {

}

//todo: need to have on instance per connection which requires a mutex in the bond manager
//todo: remove bond manager from input parameters?
func NewSmpManager(config hci.SmpConfig, bm hci.BondManager) *manager {
	return &manager{}
}

func (m *manager) SetConfig(config hci.SmpConfig) {
}

func (m *manager) SetWritePDUFunc(w func([]byte) (int, error)) {
}

func (m *manager) SetEncryptFunc(e func(info hci.BondInfo) error) {

}

func (m *manager) SetNOPFunc(f func() error) {

}

func (m *manager) InitContext(localAddr, remoteAddr []byte,
	localAddrType, remoteAddrType uint8) {
}

func (m *manager) Handle(in []byte) error {
	return errors.New("not implemented")
}

func (m *manager) Pair(authData ble.AuthData, to time.Duration) error {
	return errors.New("not implemented")
}

func (m *manager) waitResult(to time.Duration) error {
	return errors.New("not implemented")
}

func (m *manager) StartEncryption() error {
	return errors.New("not implemented")
}

//todo: implement if needed
func (m *manager) BondInfoFor(addr string) hci.BondInfo {
	return nil
}

func (m *manager) DeleteBondInfo() error {
	return errors.New("not implemented")
}

func (m *manager) LegacyPairingInfo() (bool, []byte) {
	return false, nil
}

func (m *manager) EnableEncryption(addr string) error {
	return errors.New("not implemented")
}

func (m *manager) Encrypt() error {
	return errors.New("not implemented")
}
