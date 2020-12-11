package h4

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	rxQueueSize    = 64
	txQueueSize    = 64
	defaultTimeout = time.Second * 1
)

type h4 struct {
	rwc io.ReadWriteCloser
	rmu sync.Mutex
	wmu sync.Mutex

	frame *frame

	rxQueue chan []byte
	txQueue chan []byte

	done chan int
	cmu  sync.Mutex
}

func NewSocket(addr string, connTimeout time.Duration) (io.ReadWriteCloser, error) {
	println("opening h4 socket", addr)
	c, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	// use a shorter timeout when flushing so we dont block for too long in init
	fast := time.Millisecond * 500
	rwc := &connWithTimeout{c, fast}

	// eof not ok (skt closed)
	eofAsError := true
	if err := resetAndWaitIdle(rwc, time.Second*2, eofAsError); err != nil {
		rwc.Close()
		return nil, err
	}
	println("opened", c.RemoteAddr().String())

	// set the real timeout
	rwc.timeout = connTimeout

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

func (h *h4) Read(p []byte) (int, error) {
	if !h.isOpen() {
		return 0, io.EOF
	}

	h.rmu.Lock()
	defer h.rmu.Unlock()

	var n int
	var err error

	select {
	case t := <-h.rxQueue:
		//ok
		if len(p) < len(t) {
			return 0, fmt.Errorf("buffer too small")
		}
		n = copy(p, t)

	case <-time.After(time.Second):
		return 0, nil
	}

	// check if we are still open since the read could take a while
	if !h.isOpen() {
		return 0, io.EOF
	}

	return n, errors.Wrap(err, "can't read h4")
}

func (h *h4) Write(p []byte) (int, error) {
	if !h.isOpen() {
		return 0, io.EOF
	}

	h.wmu.Lock()
	defer h.wmu.Unlock()
	n, err := h.rwc.Write(p)

	return n, errors.Wrap(err, "can't write h4")
}

func (h *h4) Close() error {
	h.cmu.Lock()
	defer h.cmu.Unlock()

	select {
	case <-h.done:
		println("h4 already closed!")
		return nil

	default:
		close(h.done)
		println("closing h4")
		h.rmu.Lock()
		err := h.rwc.Close()
		h.rmu.Unlock()

		return errors.Wrap(err, "can't close h4")
	}
}

func (h *h4) isOpen() bool {
	select {
	case <-h.done:
		println("isOpen: <-h.done, false")
		return false
	default:
		return h.rwc != nil
	}
}

func (h *h4) rxLoop(eofAsError bool) {
	defer h.Close()
	tmp := make([]byte, 512)

	for {
		select {
		case <-h.done:
			println("rxLoop killed")
			return
		default:
			if h.rwc == nil {
				println("rxLoop nil rwc")
				return
			}
		}

		// read
		n, err := h.rwc.Read(tmp)
		switch {
		case err == nil:
			// ok, process it
			h.frame.Assemble(tmp[:n])
		case os.IsTimeout(err):
			continue
		case !eofAsError && err == io.EOF:
			// trap eof, read timeout
			continue
		default:
			// uhoh!
			println(err.Error())
			return
		}
	}
}

func resetAndWaitIdle(rw io.ReadWriter, d time.Duration, eofAsError bool) error {
	to := time.Now().Add(d)

	// send dummy reset
	if _, err := rw.Write([]byte{1, 3, 12, 0}); err != nil {
		return err
	}
	<-time.After(time.Millisecond * 100)

	b := make([]byte, 2048)
	for {
		n, err := rw.Read(b)
		switch {
		case err == nil && n == 0:
			// there was nothing to read, we are done
			return nil
		case time.Now().After(to):
			// timeout, done waiting
			return fmt.Errorf("timeout waiting for idle state")
		case err == nil && n != 0:
			// got data, wait again
			continue
		case os.IsTimeout(err):
			// nothing to read, we are done
			return nil
		case !eofAsError && err == io.EOF:
			// trap eof, nothing to read, we are done
			return nil
		default:
			// real error
			return err
		}
	}
}
