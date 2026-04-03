// Package hidlcodec2 implements a HIDL Codec2 client for hardware video
// encoding via hwbinder. It connects to the software codec service
// (android.hardware.media.c2@1.2::IComponentStore/software) running in
// the media.swcodec process and talks the HIDL scatter-gather wire
// protocol.
//
// This package follows the same patterns as gralloc/hidlalloc: open
// /dev/hwbinder, look up the service via hwservicemanager, build HwParcel
// requests, and parse raw reply bytes.
package hidlcodec2

import (
	"github.com/AndroidGoLab/binder/binder"
)

// HIDL transaction codes.
//
// HIDL uses FIRST_CALL_TRANSACTION (1) + method_index within each
// interface level. The method order is defined by the .hal files.
//
// The service at @1.2::IComponentStore inherits:
//   @1.0::IComponentStore → @1.1::IComponentStore → @1.2::IComponentStore
//
// @1.0::IComponentStore transaction codes (1-based, FIRST_CALL_TRANSACTION=1):
//   1: createComponent    2: createInterface   3: listComponents
//   4: createInputSurface 5: getStructDescriptors 6: copyBuffer
//   7: getPoolClientManager 8: getConfigurable
//
// @1.1::IComponentStore adds (inherits 1-8):
//   9: createComponent_1_1
//
// @1.2::IComponentStore adds (inherits 1-9):
//   10: createComponent_1_2
//
// @1.0::IComponent transaction codes:
//   1: queue 2: flush 3: drain 4: setOutputSurface
//   5: connectToInputSurface 6: connectToOmxInputSurface
//   7: disconnectFromInputSurface 8: createBlockPool
//   9: destroyBlockPool 10: start 11: stop 12: reset
//   13: release 14: getInterface 15: asInputSink
//
// @1.0::IConfigurable transaction codes:
//   1: getId 2: getName 3: query 4: config
//   5: querySupportedParams 6: querySupportedValues
//
// @1.0::IComponentInterface transaction codes:
//   1: getConfigurable

const (
	// IComponentStore 1.0 transactions.
	transactionCreateComponent    = binder.TransactionCode(1)
	transactionCreateInterface    = binder.TransactionCode(2)
	transactionListComponents     = binder.TransactionCode(3)
	transactionGetPoolClientMgr   = binder.TransactionCode(7)
	transactionGetStoreConfigurable = binder.TransactionCode(8)

	// IComponentStore 1.1 inherits 1.0 (8 methods), adds one.
	transactionCreateComponent11 = binder.TransactionCode(9)

	// IComponentStore 1.2 inherits 1.1 (9 methods), adds one.
	transactionCreateComponent12 = binder.TransactionCode(10)

	// IComponent 1.0 transactions.
	transactionQueue        = binder.TransactionCode(1)
	transactionFlush        = binder.TransactionCode(2)
	transactionDrain        = binder.TransactionCode(3)
	transactionCreateBPool  = binder.TransactionCode(8)
	transactionDestroyBPool = binder.TransactionCode(9)
	transactionStart        = binder.TransactionCode(10)
	transactionStop         = binder.TransactionCode(11)
	transactionReset        = binder.TransactionCode(12)
	transactionRelease      = binder.TransactionCode(13)
	transactionGetInterface = binder.TransactionCode(14)

	// IConfigurable 1.0 transactions.
	transactionGetId     = binder.TransactionCode(1)
	transactionGetName   = binder.TransactionCode(2)
	transactionQuery     = binder.TransactionCode(3)
	transactionConfig    = binder.TransactionCode(4)

	// IComponentInterface 1.0 transactions.
	transactionGetConfigurable = binder.TransactionCode(1)
)
