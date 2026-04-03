//go:build linux

package kernelbinder

import "unsafe"

// binderCommand represents a binder command (BC_*) code written to the driver.
type binderCommand uint32

// Binder command (BC) codes -- written to the driver.
// These are var (not const) because the values use unsafe.Sizeof, which is
// not a constant expression in Go.
var (
	bcTransaction       = binderCommand(iow('c', 0, unsafe.Sizeof(binderTransactionData{})))
	bcReply             = binderCommand(iow('c', 1, unsafe.Sizeof(binderTransactionData{})))
	bcFreeBuffer        = binderCommand(iow('c', 3, unsafe.Sizeof(uintptr(0)))) // binder_uintptr_t
	bcIncRefs           = binderCommand(iow('c', 4, unsafe.Sizeof(uint32(0))))
	bcAcquire           = binderCommand(iow('c', 5, unsafe.Sizeof(uint32(0))))
	bcRelease           = binderCommand(iow('c', 6, unsafe.Sizeof(uint32(0))))
	bcDecRefs           = binderCommand(iow('c', 7, unsafe.Sizeof(uint32(0))))
	bcIncRefsDone       = binderCommand(iow('c', 8, binderPtrCookieSize))
	bcAcquireDone       = binderCommand(iow('c', 9, binderPtrCookieSize))
	bcEnterLooper       = binderCommand(ioc(0, 'c', 12, 0))
	bcExitLooper        = binderCommand(ioc(0, 'c', 13, 0))
	bcRequestDeathNotif = binderCommand(iow('c', 14, binderHandleCookieSize))
	bcClearDeathNotif   = binderCommand(iow('c', 15, binderHandleCookieSize))
	bcDeadBinderDone    = binderCommand(iow('c', 16, unsafe.Sizeof(uintptr(0))))
	bcTransactionSG     = binderCommand(iow('c', 17, unsafe.Sizeof(binderTransactionDataSG{})))
	bcReplySG           = binderCommand(iow('c', 18, unsafe.Sizeof(binderTransactionDataSG{})))
)

// binderTransactionDataSG extends binderTransactionData with a buffers_size
// field for scatter-gather transactions (BC_TRANSACTION_SG / BC_REPLY_SG).
type binderTransactionDataSG struct {
	binderTransactionData
	buffersSize uint64 // binder_size_t: total size of scatter-gather buffers
}

// Compile-time size assertion: binderTransactionData (64) + buffersSize (8) = 72.
var _ [72]byte = [unsafe.Sizeof(binderTransactionDataSG{})]byte{}
