//go:build linux

package kernelbinder

import (
	"context"
	"encoding/binary"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/binder/binder"
	aidlerrors "github.com/xaionaro-go/binder/errors"
	"github.com/xaionaro-go/binder/parcel"
	"golang.org/x/sys/unix"
)

// RegisterReceiver associates a TransactionReceiver with a unique cookie
// and starts the read loop if it has not been started yet.
// Returns the cookie that identifies this receiver in incoming BR_TRANSACTION events.
func (d *Driver) RegisterReceiver(
	ctx context.Context,
	receiver binder.TransactionReceiver,
) uintptr {
	logger.Tracef(ctx, "RegisterReceiver")

	cookie := d.nextCookie.Add(1)

	d.receiversMu.Lock()
	d.receivers[cookie] = receiver
	d.receiversMu.Unlock()

	d.readLoopOnce.Do(func() {
		// Use a plain goroutine because observability.Go is not available
		// in this module. TODO: switch to observability.Go when available.
		go d.runReadLoop(ctx)
	})

	logger.Tracef(ctx, "/RegisterReceiver: cookie=%d", cookie)
	return cookie
}

// UnregisterReceiver removes a TransactionReceiver by its cookie.
func (d *Driver) UnregisterReceiver(
	ctx context.Context,
	cookie uintptr,
) {
	logger.Tracef(ctx, "UnregisterReceiver")
	defer func() { logger.Tracef(ctx, "/UnregisterReceiver") }()

	d.receiversMu.Lock()
	delete(d.receivers, cookie)
	d.receiversMu.Unlock()
}

// lookupReceiver returns the TransactionReceiver for the given cookie,
// or nil if no receiver is registered.
func (d *Driver) lookupReceiver(
	cookie uintptr,
) binder.TransactionReceiver {
	d.receiversMu.RLock()
	defer d.receiversMu.RUnlock()
	return d.receivers[cookie]
}

// runReadLoop is the background goroutine that reads BR_* events
// from the binder driver and dispatches incoming transactions.
func (d *Driver) runReadLoop(
	ctx context.Context,
) {
	logger.Tracef(ctx, "runReadLoop")
	defer func() {
		logger.Tracef(ctx, "/runReadLoop")
		close(d.readLoopDone)
	}()

	// Pin to OS thread: binder routes replies by thread ID.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Tell the kernel this thread accepts incoming transactions.
	enterBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(enterBuf[0:4], bcEnterLooper)

	d.mu.Lock()
	err := d.writeCommand(ctx, enterBuf)
	d.mu.Unlock()
	if err != nil {
		logger.Errorf(ctx, "failed to send BC_ENTER_LOOPER: %v", err)
		return
	}

	readBuf := make([]byte, readBufferSize)
	for {
		select {
		case <-ctx.Done():
			d.sendExitLooper(ctx)
			return
		default:
		}

		bwr := binderWriteRead{
			readSize:   uint64(len(readBuf)),
			readBuffer: uint64(uintptr(unsafe.Pointer(&readBuf[0]))),
		}

		d.mu.Lock()
		var errno unix.Errno
		for {
			_, _, errno = unix.Syscall(
				unix.SYS_IOCTL,
				uintptr(d.fd),
				binderWriteReadIoctl,
				uintptr(unsafe.Pointer(&bwr)),
			)
			if errno == unix.EINTR {
				continue
			}
			break
		}
		d.mu.Unlock()

		if errno != 0 {
			// If the fd was closed (e.g. during shutdown), stop gracefully.
			if errno == unix.EBADF {
				logger.Debugf(ctx, "read loop: fd closed, exiting")
				return
			}
			logger.Errorf(ctx, "read loop ioctl error: %v", errno)
			return
		}

		if bwr.readConsumed == 0 {
			continue
		}

		d.dispatchReadBuffer(ctx, readBuf[:bwr.readConsumed])
	}
}

// sendExitLooper notifies the kernel that this thread is leaving the looper.
func (d *Driver) sendExitLooper(
	ctx context.Context,
) {
	exitBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(exitBuf[0:4], bcExitLooper)

	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.writeCommand(ctx, exitBuf); err != nil {
		logger.Warnf(ctx, "failed to send BC_EXIT_LOOPER: %v", err)
	}
}

// dispatchReadBuffer processes BR_* codes from the read buffer,
// handling incoming transactions and reference management commands.
func (d *Driver) dispatchReadBuffer(
	ctx context.Context,
	buf []byte,
) {
	logger.Tracef(ctx, "dispatchReadBuffer")
	defer func() { logger.Tracef(ctx, "/dispatchReadBuffer") }()

	offset := 0
	txnSize := int(unsafe.Sizeof(binderTransactionData{}))
	ptrCookieSize := int(binderPtrCookieSize)

	for offset < len(buf) {
		if offset+4 > len(buf) {
			logger.Warnf(ctx, "read loop: truncated BR code at offset %d", offset)
			return
		}

		cmd := binary.LittleEndian.Uint32(buf[offset:])
		offset += 4

		switch cmd {
		case brNoop:
			logger.Debugf(ctx, "read loop: BR_NOOP")

		case brTransactionComplete:
			logger.Debugf(ctx, "read loop: BR_TRANSACTION_COMPLETE")

		case brTransaction:
			logger.Debugf(ctx, "read loop: BR_TRANSACTION")
			if offset+txnSize > len(buf) {
				logger.Warnf(ctx, "read loop: truncated BR_TRANSACTION at offset %d", offset)
				return
			}

			var txn binderTransactionData
			copyBytesToStruct(unsafe.Pointer(&txn), buf[offset:], unsafe.Sizeof(txn))
			offset += txnSize

			d.handleIncomingTransaction(ctx, &txn)

		case brReply:
			// The read loop should not normally receive BR_REPLY;
			// those are handled by the synchronous Transact path.
			logger.Warnf(ctx, "read loop: unexpected BR_REPLY")
			offset += txnSize

		case brIncRefs:
			logger.Debugf(ctx, "read loop: BR_INCREFS")
			if offset+ptrCookieSize > len(buf) {
				logger.Warnf(ctx, "read loop: truncated BR_INCREFS at offset %d", offset)
				return
			}
			d.handleRefCommand(ctx, bcIncRefsDone, buf[offset:offset+ptrCookieSize])
			offset += ptrCookieSize

		case brAcquire:
			logger.Debugf(ctx, "read loop: BR_ACQUIRE")
			if offset+ptrCookieSize > len(buf) {
				logger.Warnf(ctx, "read loop: truncated BR_ACQUIRE at offset %d", offset)
				return
			}
			d.handleRefCommand(ctx, bcAcquireDone, buf[offset:offset+ptrCookieSize])
			offset += ptrCookieSize

		case brRelease:
			logger.Debugf(ctx, "read loop: BR_RELEASE")
			if offset+ptrCookieSize > len(buf) {
				logger.Warnf(ctx, "read loop: truncated BR_RELEASE at offset %d", offset)
				return
			}
			// No acknowledgment needed; reference counting is a no-op for now.
			offset += ptrCookieSize

		case brDecrefs:
			logger.Debugf(ctx, "read loop: BR_DECREFS")
			if offset+ptrCookieSize > len(buf) {
				logger.Warnf(ctx, "read loop: truncated BR_DECREFS at offset %d", offset)
				return
			}
			// No acknowledgment needed; reference counting is a no-op for now.
			offset += ptrCookieSize

		case brDeadReply:
			logger.Debugf(ctx, "read loop: BR_DEAD_REPLY")

		case brFailedReply:
			logger.Debugf(ctx, "read loop: BR_FAILED_REPLY")

		case brError:
			if offset+4 > len(buf) {
				logger.Warnf(ctx, "read loop: truncated BR_ERROR at offset %d", offset)
				return
			}
			errCode := int32(binary.LittleEndian.Uint32(buf[offset:]))
			offset += 4
			logger.Errorf(ctx, "read loop: BR_ERROR %d", errCode)

		case brSpawnLooper:
			logger.Debugf(ctx, "read loop: BR_SPAWN_LOOPER (ignored)")

		default:
			logger.Warnf(ctx, "read loop: unknown BR code 0x%08x at offset %d", cmd, offset-4)
			return
		}
	}
}

// handleRefCommand acknowledges a BR_INCREFS or BR_ACQUIRE with the
// corresponding BC_INCREFS_DONE or BC_ACQUIRE_DONE response.
func (d *Driver) handleRefCommand(
	ctx context.Context,
	doneCmd uint32,
	ptrCookieBuf []byte,
) {
	// The payload is binder_ptr_cookie: two pointer-sized values (ptr + cookie).
	// We echo them back in BC_INCREFS_DONE / BC_ACQUIRE_DONE.
	ackBuf := make([]byte, 4+len(ptrCookieBuf))
	binary.LittleEndian.PutUint32(ackBuf[0:4], doneCmd)
	copy(ackBuf[4:], ptrCookieBuf)

	d.mu.Lock()
	err := d.writeCommand(ctx, ackBuf)
	d.mu.Unlock()

	if err != nil {
		logger.Warnf(ctx, "read loop: failed to send ref acknowledgment 0x%08x: %v", doneCmd, err)
	}
}

// handleIncomingTransaction processes a BR_TRANSACTION by looking up the
// registered receiver, calling OnTransaction, sending BC_REPLY, and
// freeing the transaction buffer.
func (d *Driver) handleIncomingTransaction(
	ctx context.Context,
	txn *binderTransactionData,
) {
	logger.Tracef(ctx, "handleIncomingTransaction")
	defer func() { logger.Tracef(ctx, "/handleIncomingTransaction") }()

	cookie := uintptr(txn.cookie)
	code := binder.TransactionCode(txn.code)
	isOneway := binder.TransactionFlags(txn.flags)&binder.FlagOneway != 0

	// Copy transaction data from the mmap'd region.
	var dataBytes []byte
	if txn.dataSize > 0 {
		dataBytes = d.copyFromMapped(txn.dataBuffer, txn.dataSize)
	}

	// Free the mmap'd buffer as soon as we have copied the data.
	if txn.dataSize > 0 {
		d.mu.Lock()
		err := d.freeBuffer(ctx, txn.dataBuffer)
		d.mu.Unlock()
		if err != nil {
			logger.Warnf(ctx, "failed to free transaction buffer: %v", err)
		}
	}

	receiver := d.lookupReceiver(cookie)
	if receiver == nil {
		logger.Warnf(ctx, "no receiver registered for cookie %d", cookie)
		if !isOneway {
			d.sendReply(ctx, parcel.New())
		}
		return
	}

	dataParcel := parcel.FromBytes(dataBytes)
	replyParcel, err := receiver.OnTransaction(ctx, code, dataParcel)
	if err != nil {
		logger.Warnf(ctx, "OnTransaction(code=%d) failed: %v", code, err)
		if !isOneway {
			// Send an empty reply on error so the caller does not hang.
			d.sendReply(ctx, parcel.New())
		}
		return
	}

	// For one-way transactions the kernel does not expect a reply.
	if isOneway {
		return
	}

	if replyParcel == nil {
		replyParcel = parcel.New()
	}

	d.sendReply(ctx, replyParcel)
}

// sendReply sends a BC_REPLY with the given parcel data.
func (d *Driver) sendReply(
	ctx context.Context,
	reply *parcel.Parcel,
) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.Reply(ctx, reply); err != nil {
		logger.Warnf(ctx, "failed to send BC_REPLY: %v", err)
	}
}

// Reply sends BC_REPLY with the given parcel data back to the
// transaction originator. Must be called with d.mu held.
func (d *Driver) Reply(
	ctx context.Context,
	reply *parcel.Parcel,
) (_err error) {
	logger.Tracef(ctx, "Reply")
	defer func() { logger.Tracef(ctx, "/Reply: %v", _err) }()

	replyData := reply.Data()
	objects := reply.Objects()

	var dataPtr, offsetsPtr uint64
	if len(replyData) > 0 {
		dataPtr = uint64(uintptr(unsafe.Pointer(&replyData[0])))
	}

	var offsetsBuf []byte
	if len(objects) > 0 {
		offsetsBuf = make([]byte, len(objects)*8)
		for i, off := range objects {
			binary.LittleEndian.PutUint64(offsetsBuf[i*8:], off)
		}
		offsetsPtr = uint64(uintptr(unsafe.Pointer(&offsetsBuf[0])))
	}

	txn := binderTransactionData{
		code:          0,
		flags:         0,
		dataSize:      uint64(len(replyData)),
		offsetsSize:   uint64(len(objects) * 8),
		dataBuffer:    dataPtr,
		offsetsBuffer: offsetsPtr,
	}

	txnSize := unsafe.Sizeof(txn)
	writeBuf := make([]byte, 4+txnSize)
	binary.LittleEndian.PutUint32(writeBuf[0:4], bcReply)
	copyStructToBytes(writeBuf[4:], unsafe.Pointer(&txn), txnSize)

	readBuf := make([]byte, readBufferSize)

	bwr := binderWriteRead{
		writeSize:   uint64(len(writeBuf)),
		writeBuffer: uint64(uintptr(unsafe.Pointer(&writeBuf[0]))),
		readSize:    uint64(len(readBuf)),
		readBuffer:  uint64(uintptr(unsafe.Pointer(&readBuf[0]))),
	}

	for {
		_, _, errno := unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(d.fd),
			binderWriteReadIoctl,
			uintptr(unsafe.Pointer(&bwr)),
		)
		switch errno {
		case 0:
		case unix.EINTR:
			bwr.writeSize = 0
			continue
		default:
			return fmt.Errorf("Reply: %w", &aidlerrors.BinderError{
				Op:  "ioctl(BINDER_WRITE_READ)",
				Err: errno,
			})
		}
		break
	}

	return nil
}
