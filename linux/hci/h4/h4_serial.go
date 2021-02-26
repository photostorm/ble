// +build bleserial

package h4

import (
	"io"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

func DefaultSerialOptions() serial.OpenOptions {
	return serial.OpenOptions{
		PortName:              "/dev/ttyACM0",
		BaudRate:              115200,
		DataBits:              8,
		ParityMode:            serial.PARITY_NONE,
		StopBits:              1,
		RTSCTSFlowControl:     true,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
	}
}

func NewSerial(portName string) (io.ReadWriteCloser, error) {
	opts := DefaultSerialOptions()
	opts.PortName = portName

	// force these
	opts.MinimumReadSize = 0
	opts.InterCharacterTimeout = 100

	println("opening h4 uart ", opts.PortName)
	rwc, err := serial.Open(opts)
	if err != nil {
		return nil, err
	}

	// eof is ok (read timeout)
	eofAsError := false
	if err := resetAndWaitIdle(rwc, time.Second*2, eofAsError); err != nil {
		_ = rwc.Close()
		return nil, err
	}

	h := &h4{
		rwc:     rwc,
		done:    make(chan int),
		rxQueue: make(chan []byte, rxQueueSize),
		txQueue: make(chan []byte, txQueueSize),
	}
	h.frame = newFrame(h.rxQueue)

	go h.rxLoop(eofAsError)

	return h, nil
}
