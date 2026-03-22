//go:build linux

package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/facebookincubator/go-belt/tool/logger"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// Daemon is a device-side TCP server that forwards binder transactions
// from a host-side proxy to the Android binder driver.
type Daemon struct {
	listenAddr string
	driver     *kernelbinder.Driver
	transport  *versionaware.Transport
	sm         *servicemanager.ServiceManager

	// descriptorCache maps AIDL interface descriptors to their IBinder handles.
	descriptorCache   map[string]binder.IBinder
	descriptorCacheMu sync.RWMutex
}

// NewDaemon opens the binder driver and creates a ServiceManager.
func NewDaemon(
	ctx context.Context,
	opts ...DaemonOption,
) (_ *Daemon, _err error) {
	logger.Tracef(ctx, "NewDaemon")
	defer func() { logger.Tracef(ctx, "/NewDaemon: %v", _err) }()

	cfg := DaemonOptions(opts).config()

	driver, err := kernelbinder.Open(ctx, binder.WithMapSize(cfg.MapSize))
	if err != nil {
		return nil, fmt.Errorf("opening binder driver: %w", err)
	}
	defer func() {
		if _err != nil {
			_ = driver.Close(ctx)
		}
	}()

	transport, err := versionaware.NewTransport(ctx, driver, 0)
	if err != nil {
		return nil, fmt.Errorf("creating version-aware transport: %w", err)
	}

	sm := servicemanager.New(transport)

	return &Daemon{
		listenAddr:      cfg.ListenAddr,
		driver:          driver,
		transport:       transport,
		sm:              sm,
		descriptorCache: make(map[string]binder.IBinder),
	}, nil
}

// Serve accepts TCP connections and handles binder transactions.
// Blocks until ctx is cancelled.
func (d *Daemon) Serve(
	ctx context.Context,
) (_err error) {
	logger.Tracef(ctx, "Serve")
	defer func() { logger.Tracef(ctx, "/Serve: %v", _err) }()

	ln, err := net.Listen("tcp", d.listenAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", d.listenAddr, err)
	}
	defer ln.Close()

	logger.Debugf(ctx, "daemon listening on %s", ln.Addr())

	// Close the listener when the context is cancelled so Accept unblocks.
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return fmt.Errorf("accepting connection: %w", err)
			}
		}

		logger.Debugf(ctx, "accepted connection from %s", conn.RemoteAddr())
		go d.handleConnection(ctx, conn)
	}
}

// Close releases the binder driver.
func (d *Daemon) Close(
	ctx context.Context,
) error {
	return d.driver.Close(ctx)
}

// handleConnection processes requests on a single TCP connection.
func (d *Daemon) handleConnection(
	ctx context.Context,
	conn net.Conn,
) {
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		descriptor, code, flags, data, err := ReadRequest(conn)
		if err != nil {
			logger.Debugf(ctx, "reading request: %v", err)
			return
		}

		logger.Debugf(ctx, "request: descriptor=%s code=%d flags=%d data_len=%d", descriptor, code, flags, len(data))

		replyData, statusCode := d.executeTransaction(ctx, descriptor, code, flags, data)

		if err := WriteResponse(conn, statusCode, replyData); err != nil {
			logger.Debugf(ctx, "writing response: %v", err)
			return
		}
	}
}

// statusTransactFailed is the status code returned when the transaction
// cannot be executed (service not found, transact error, etc.).
const statusTransactFailed = 1

// executeTransaction resolves the service for the descriptor and
// executes the binder transaction.
func (d *Daemon) executeTransaction(
	ctx context.Context,
	descriptor string,
	code uint32,
	flags uint32,
	data []byte,
) (replyData []byte, statusCode uint32) {
	svc, err := d.resolveService(ctx, descriptor)
	if err != nil {
		logger.Warnf(ctx, "resolving service for %q: %v", descriptor, err)
		return nil, statusTransactFailed
	}

	p := parcel.FromBytes(data)
	reply, err := svc.Transact(
		ctx,
		binder.TransactionCode(code),
		binder.TransactionFlags(flags),
		p,
	)
	if err != nil {
		logger.Warnf(ctx, "transact %s code=%d: %v", descriptor, code, err)
		return nil, statusTransactFailed
	}
	defer reply.Recycle()

	return reply.Data(), 0
}

// resolveService finds the IBinder for a given AIDL interface descriptor.
// Uses a cache to avoid repeated lookups.
func (d *Daemon) resolveService(
	ctx context.Context,
	descriptor string,
) (binder.IBinder, error) {
	// Fast path: cached.
	d.descriptorCacheMu.RLock()
	svc, ok := d.descriptorCache[descriptor]
	d.descriptorCacheMu.RUnlock()
	if ok {
		return svc, nil
	}

	// Slow path: scan all services to find the one with this descriptor.
	d.descriptorCacheMu.Lock()
	defer d.descriptorCacheMu.Unlock()

	// Double-check after acquiring write lock.
	if svc, ok := d.descriptorCache[descriptor]; ok {
		return svc, nil
	}

	// Special case: the ServiceManager descriptor is always handle 0.
	if descriptor == "android.os.IServiceManager" {
		smBinder := binder.NewProxyBinder(d.transport, binder.DefaultCallerIdentity(), 0)
		d.descriptorCache[descriptor] = smBinder
		return smBinder, nil
	}

	services, err := d.sm.ListServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	for _, name := range services {
		// Skip services we have already resolved.
		svcBinder, err := d.sm.GetService(ctx, name)
		if err != nil {
			logger.Debugf(ctx, "getting service %q: %v", name, err)
			continue
		}

		svcDescriptor, err := queryDescriptor(ctx, svcBinder)
		if err != nil {
			logger.Debugf(ctx, "querying descriptor for %q: %v", name, err)
			continue
		}

		d.descriptorCache[svcDescriptor] = svcBinder

		if svcDescriptor == descriptor {
			return svcBinder, nil
		}
	}

	return nil, fmt.Errorf("no service found for descriptor %q", descriptor)
}

// queryDescriptor sends INTERFACE_TRANSACTION to an IBinder to learn
// its AIDL interface descriptor string.
func queryDescriptor(
	ctx context.Context,
	svc binder.IBinder,
) (_ string, _err error) {
	data := parcel.New()
	defer data.Recycle()

	reply, err := svc.Transact(ctx, binder.InterfaceTransaction, 0, data)
	if err != nil {
		return "", fmt.Errorf("INTERFACE_TRANSACTION: %w", err)
	}
	defer reply.Recycle()

	desc, err := reply.ReadString16()
	if err != nil {
		return "", fmt.Errorf("reading descriptor: %w", err)
	}

	return desc, nil
}
