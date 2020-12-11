package dev

import (
	"github.com/photostorm/ble"
	"github.com/photostorm/ble/linux"
)

// DefaultDevice ...
func DefaultDevice(opts ...ble.Option) (d ble.Device, err error) {
	return linux.NewDevice(opts...)
}
