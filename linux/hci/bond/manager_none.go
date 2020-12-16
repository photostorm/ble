// +build !blesmp

package bond

import (
	"errors"

	"github.com/photostorm/ble/linux/hci"
)

type manager struct {

}

func NewBondManager(bondFilePath string) hci.BondManager {
	return &manager{}
}

//todo: is this function really needed?
func (m *manager) Exists(addr string) bool {
	return false
}

func (m *manager) Find(addr string) (hci.BondInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *manager) Save(addr string, bond hci.BondInfo) error {
	return errors.New("not implemented")
}

func (m *manager) Delete(addr string) error {
	return errors.New("not implemented")
}