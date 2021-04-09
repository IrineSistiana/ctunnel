package ctunnel

import (
	"io"
	"net"
	"sync"
	"time"
)

// OpenTunnel opens a tunnel between a and b.
// It returns the first err encountered.
// a and b will be closed by OpenTunnel.
func OpenTunnel(a, b net.Conn, timeout time.Duration) error {
	t := newTunnel(a, b, timeout)
	t.openOneWayTunnel(a, b)
	t.openOneWayTunnel(b, a)
	return t.wait()
}

type tunnel struct {
	a, b    net.Conn
	timeout time.Duration

	wg        sync.WaitGroup
	closeOnce sync.Once
	closeErr  error
}

func newTunnel(a, b net.Conn, timeout time.Duration) *tunnel {
	return &tunnel{a: a, b: b, timeout: timeout}
}

func (t *tunnel) close(err error) {
	t.closeOnce.Do(func() {
		t.a.Close()
		t.b.Close()
		t.closeErr = err
	})
}

func (t *tunnel) openOneWayTunnel(dst, src net.Conn) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		_, err := t.copyBuffer(dst, src)
		t.close(err)
	}()
}

func (t *tunnel) wait() error {
	t.wg.Wait()
	return t.closeErr
}

func (t *tunnel) copyBuffer(dst net.Conn, src net.Conn) (written int64, err error) {
	buf := acquireIOBuf()
	defer releaseIOBuf(buf)

	for {
		src.SetDeadline(time.Now().Add(t.timeout))
		nr, er := src.Read(buf)
		if nr > 0 {
			dst.SetDeadline(time.Now().Add(t.timeout))
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

var (
	ioCopyBuffPool = &sync.Pool{New: func() interface{} {
		return make([]byte, 32*1024)
	}}
)

func acquireIOBuf() []byte {
	return ioCopyBuffPool.Get().([]byte)
}

func releaseIOBuf(b []byte) {
	ioCopyBuffPool.Put(b)
}
