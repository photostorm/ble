// +build !bleserial

package h4

import (
	"errors"
	"io"
)

func NewSerial(portName string) (io.ReadWriteCloser, error) {
	return nil, errors.New("not enabled")
}
