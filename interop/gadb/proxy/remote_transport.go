package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/facebookincubator/go-belt/tool/logger"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
)

// RemoteTransport sends binder transactions over TCP to a device-side
// Daemon. It implements a Transact method compatible with generated
// proxy code.
type RemoteTransport struct {
	conn net.Conn
	mu   sync.Mutex
}

// NewRemoteTransport connects to the daemon at the given TCP address.
func NewRemoteTransport(
	addr string,
) (_ *RemoteTransport, _err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connecting to daemon at %s: %w", addr, err)
	}

	return &RemoteTransport{conn: conn}, nil
}

// Transact sends a binder transaction request to the remote daemon
// and returns the reply parcel.
func (rt *RemoteTransport) Transact(
	ctx context.Context,
	descriptor string,
	code uint32,
	flags uint32,
	data *parcel.Parcel,
) (_ *parcel.Parcel, _err error) {
	logger.Tracef(ctx, "RemoteTransport.Transact(%s, code=%d)", descriptor, code)
	defer func() { logger.Tracef(ctx, "/RemoteTransport.Transact: %v", _err) }()

	var rawData []byte
	if data != nil {
		rawData = data.Data()
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	if err := WriteRequest(rt.conn, descriptor, code, flags, rawData); err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	statusCode, replyData, err := ReadResponse(rt.conn)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if statusCode != 0 {
		return nil, &TransactError{
			Descriptor: descriptor,
			Code:       binder.TransactionCode(code),
			StatusCode: statusCode,
		}
	}

	return parcel.FromBytes(replyData), nil
}

// Close closes the TCP connection to the daemon.
func (rt *RemoteTransport) Close() error {
	return rt.conn.Close()
}

// TransactError indicates that the daemon returned a non-zero status code.
type TransactError struct {
	Descriptor string
	Code       binder.TransactionCode
	StatusCode uint32
}

func (e *TransactError) Error() string {
	return fmt.Sprintf(
		"remote transact %s code=%d failed: daemon status %d",
		e.Descriptor, e.Code, e.StatusCode,
	)
}
