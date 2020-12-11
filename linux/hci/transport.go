package hci

import (
	"errors"
	"io"
	"time"

	"github.com/photostorm/ble/linux/hci/h4"
	"github.com/photostorm/ble/linux/hci/socket"
)

type transportHci struct {
	id int
}

type transportH4Socket struct {
	addr    string
	timeout time.Duration
}

type transportH4Uart struct {
	path string
}

type transport struct {
	hci      *transportHci
	h4uart   *transportH4Uart
	h4socket *transportH4Socket
}

func getTransport(t transport) (io.ReadWriteCloser, error) {
	switch {
	case t.hci != nil:
		return socket.NewSocket(t.hci.id)
	case t.h4socket != nil:
		return h4.NewSocket(t.h4socket.addr, t.h4socket.timeout)
	case t.h4uart != nil:
		return h4.NewSerial(t.h4uart.path)
	default:
		return nil, errors.New("no valid transport found")
	}
}
