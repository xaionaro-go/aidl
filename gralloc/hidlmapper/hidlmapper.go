// Package hidlmapper implements the HIDL IMapper@3.0 client for triggering
// host GPU readback on goldfish/ranchu emulator buffers via hwbinder.
//
// The ranchu mapper's lock() internally calls rcReadColorBuffer to transfer
// pixel data from the host GPU into the goldfish address space memory.
// When a buffer is mmap'd (via /dev/goldfish_address_space), calling
// lock() populates the shared memory visible to all processes that have
// mmap'd the same region. The actual pointer returned by lock() is only
// valid in the mapper's process, but the underlying data is in shared
// device memory accessible through our mmap.
//
// IMapper@3.0 is a standalone interface (no parent inheritance) with
// transaction codes starting at FIRST_CALL_TRANSACTION (1):
//
//	1: createDescriptor
//	2: importBuffer
//	3: freeBuffer
//	4: validateBufferSize
//	5: getTransportSize
//	6: lock
//	7: lockYCbCr
//	8: unlock
//	9: isSupported
package hidlmapper

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/hwparcel"
	"github.com/AndroidGoLab/binder/hwservicemanager"
	"github.com/AndroidGoLab/binder/logger"
)

const (
	// mapperFQName is the HIDL fully-qualified interface name.
	mapperFQName = "android.hardware.graphics.mapper@3.0::IMapper"

	// mapperInstance is the default service instance name.
	mapperInstance = "default"

	// Transaction codes for IMapper@3.0 (standalone, no parent).
	transactionImportBuffer = binder.TransactionCode(2)
	transactionFreeBuffer   = binder.TransactionCode(3)
	transactionLock         = binder.TransactionCode(6)
	transactionLockYCbCr    = binder.TransactionCode(7)
	transactionUnlock       = binder.TransactionCode(8)

	// cpuUsageReadOften is GRALLOC_USAGE_SW_READ_OFTEN (0x3).
	cpuUsageReadOften = uint64(0x3)
)

// Mapper is a HIDL IMapper@3.0 client connected via hwbinder.
type Mapper struct {
	transport    binder.Transport
	mapperHandle uint32
}

// New looks up the IMapper@3.0 service via hwservicemanager and returns
// a connected Mapper client.
func New(
	ctx context.Context,
	transport binder.Transport,
) (_ *Mapper, _err error) {
	logger.Tracef(ctx, "hidlmapper.New")
	defer func() { logger.Tracef(ctx, "/hidlmapper.New: %v", _err) }()

	sm := hwservicemanager.New(transport)
	handle, err := sm.GetService(ctx, mapperFQName, mapperInstance)
	if err != nil {
		return nil, fmt.Errorf("getting mapper service: %w", err)
	}

	return &Mapper{
		transport:    transport,
		mapperHandle: handle,
	}, nil
}

// ImportBuffer imports a raw gralloc buffer handle into the mapper.
// Returns an opaque buffer token for use with Lock/Unlock/FreeBuffer.
//
// Wire format (request):
//
//	interface token
//	hidl_handle: top-level buffer object (16 bytes)
//	  -> child: native_handle_t data
//	  -> binder_fd_array_object for FDs
//
// Wire format (response):
//
//	int32 error (0 = NONE)
//	padding (4 bytes)
//	uint64 buffer (opaque token)
func (m *Mapper) ImportBuffer(
	ctx context.Context,
	fds []int32,
	ints []int32,
) (_ uint64, _err error) {
	logger.Tracef(ctx, "hidlmapper.ImportBuffer(numFds=%d, numInts=%d)", len(fds), len(ints))
	defer func() { logger.Tracef(ctx, "/hidlmapper.ImportBuffer: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(mapperFQName)
	hp.WriteNativeHandle(fds, ints)

	reply, err := hwservicemanager.TransactHidl(ctx, m.transport, m.mapperHandle, transactionImportBuffer, hp)
	if err != nil {
		return 0, fmt.Errorf("importBuffer transaction: %w", err)
	}
	defer reply.Recycle()

	data := reply.Data()
	if len(data) < 16 {
		return 0, fmt.Errorf("importBuffer response too short: %d bytes", len(data))
	}

	mapperErr := int32(binary.LittleEndian.Uint32(data[0:4]))
	if mapperErr != 0 {
		return 0, fmt.Errorf("importBuffer error: %d", mapperErr)
	}

	// The buffer token is at offset 8 (after 4-byte error + 4-byte padding).
	bufferToken := binary.LittleEndian.Uint64(data[8:16])
	if bufferToken == 0 {
		return 0, fmt.Errorf("importBuffer returned null buffer")
	}

	return bufferToken, nil
}

// FreeBuffer releases a previously imported buffer.
//
// Wire format (request):
//
//	interface token
//	uint64 buffer (opaque token, inline)
//
// Wire format (response):
//
//	int32 error
func (m *Mapper) FreeBuffer(
	ctx context.Context,
	buffer uint64,
) (_err error) {
	logger.Tracef(ctx, "hidlmapper.FreeBuffer(buffer=0x%x)", buffer)
	defer func() { logger.Tracef(ctx, "/hidlmapper.FreeBuffer: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(mapperFQName)
	hp.WriteUint64(buffer)

	reply, err := hwservicemanager.TransactHidl(ctx, m.transport, m.mapperHandle, transactionFreeBuffer, hp)
	if err != nil {
		return fmt.Errorf("freeBuffer transaction: %w", err)
	}
	defer reply.Recycle()

	data := reply.Data()
	if len(data) < 4 {
		return fmt.Errorf("freeBuffer response too short: %d bytes", len(data))
	}

	mapperErr := int32(binary.LittleEndian.Uint32(data[0:4]))
	if mapperErr != 0 {
		return fmt.Errorf("freeBuffer error: %d", mapperErr)
	}

	return nil
}

// Lock triggers a CPU read lock on the buffer. On goldfish/ranchu emulators,
// this causes the mapper to call rcReadColorBuffer, transferring pixel data
// from the host GPU into the shared goldfish address space memory. The
// returned data pointer is only valid in the mapper's process, but the
// pixel data is now available in any mmap of the same goldfish region.
//
// Wire format (request):
//
//	interface token
//	uint64 buffer (opaque token, inline)
//	uint64 cpuUsage (inline)
//	Rect accessRegion: int32 left, top, width, height (inline)
//	hidl_handle acquireFence (null = no fence)
//
// Wire format (response):
//
//	int32 error
//	padding (4 bytes)
//	uint64 data pointer (ignored -- only valid in mapper's process)
//	int32 bytesPerPixel
//	int32 bytesPerStride
func (m *Mapper) Lock(
	ctx context.Context,
	buffer uint64,
	width int32,
	height int32,
) (_err error) {
	logger.Tracef(ctx, "hidlmapper.Lock(buffer=0x%x, %dx%d)", buffer, width, height)
	defer func() { logger.Tracef(ctx, "/hidlmapper.Lock: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(mapperFQName)
	hp.WriteUint64(buffer)
	hp.WriteUint64(cpuUsageReadOften)

	// Rect accessRegion: left, top, width, height.
	hp.WriteInt32(0)
	hp.WriteInt32(0)
	hp.WriteInt32(width)
	hp.WriteInt32(height)

	// acquireFence: null handle (no fence).
	hp.WriteNullNativeHandle()

	reply, err := hwservicemanager.TransactHidl(ctx, m.transport, m.mapperHandle, transactionLock, hp)
	if err != nil {
		return fmt.Errorf("lock transaction: %w", err)
	}
	defer reply.Recycle()

	data := reply.Data()
	if len(data) < 4 {
		return fmt.Errorf("lock response too short: %d bytes", len(data))
	}

	mapperErr := int32(binary.LittleEndian.Uint32(data[0:4]))
	if mapperErr != 0 {
		return fmt.Errorf("lock error: %d", mapperErr)
	}

	return nil
}

// LockYCbCr triggers a CPU read lock for a YCbCr buffer. Like Lock, this
// causes the mapper to fetch pixel data from the host GPU into shared
// memory. The returned YCbCr layout pointers are only valid in the
// mapper's process.
//
// Wire format (request): same as Lock.
//
// Wire format (response):
//
//	int32 error
//	padding (4 bytes)
//	YCbCrLayout: y(8) + cb(8) + cr(8) + yStride(4) + cStride(4) + chromaStep(4) + pad(4)
func (m *Mapper) LockYCbCr(
	ctx context.Context,
	buffer uint64,
	width int32,
	height int32,
) (_err error) {
	logger.Tracef(ctx, "hidlmapper.LockYCbCr(buffer=0x%x, %dx%d)", buffer, width, height)
	defer func() { logger.Tracef(ctx, "/hidlmapper.LockYCbCr: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(mapperFQName)
	hp.WriteUint64(buffer)
	hp.WriteUint64(cpuUsageReadOften)

	// Rect accessRegion: left, top, width, height.
	hp.WriteInt32(0)
	hp.WriteInt32(0)
	hp.WriteInt32(width)
	hp.WriteInt32(height)

	// acquireFence: null handle (no fence).
	hp.WriteNullNativeHandle()

	reply, err := hwservicemanager.TransactHidl(ctx, m.transport, m.mapperHandle, transactionLockYCbCr, hp)
	if err != nil {
		return fmt.Errorf("lockYCbCr transaction: %w", err)
	}
	defer reply.Recycle()

	data := reply.Data()
	if len(data) < 4 {
		return fmt.Errorf("lockYCbCr response too short: %d bytes", len(data))
	}

	mapperErr := int32(binary.LittleEndian.Uint32(data[0:4]))
	if mapperErr != 0 {
		return fmt.Errorf("lockYCbCr error: %d", mapperErr)
	}

	return nil
}

// Unlock unlocks a previously locked buffer.
//
// Wire format (request):
//
//	interface token
//	uint64 buffer (opaque token, inline)
//
// Wire format (response):
//
//	int32 error
//	hidl_handle releaseFence (ignored)
func (m *Mapper) Unlock(
	ctx context.Context,
	buffer uint64,
) (_err error) {
	logger.Tracef(ctx, "hidlmapper.Unlock(buffer=0x%x)", buffer)
	defer func() { logger.Tracef(ctx, "/hidlmapper.Unlock: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(mapperFQName)
	hp.WriteUint64(buffer)

	reply, err := hwservicemanager.TransactHidl(ctx, m.transport, m.mapperHandle, transactionUnlock, hp)
	if err != nil {
		return fmt.Errorf("unlock transaction: %w", err)
	}
	defer reply.Recycle()

	data := reply.Data()
	if len(data) < 4 {
		return fmt.Errorf("unlock response too short: %d bytes", len(data))
	}

	mapperErr := int32(binary.LittleEndian.Uint32(data[0:4]))
	if mapperErr != 0 {
		return fmt.Errorf("unlock error: %d", mapperErr)
	}

	return nil
}
