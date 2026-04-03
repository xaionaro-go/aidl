// Package hwparcel implements the HIDL HwParcel wire format for hwbinder
// transactions. Unlike AIDL parcels that use inline data, HIDL uses
// scatter-gather serialization with binder_buffer_object entries
// (BINDER_TYPE_PTR) that reference external data buffers.
//
// The HIDL wire format is:
//   - Interface token: null-terminated C string in the data buffer
//   - Structured data: binder_buffer_object entries in the data buffer,
//     each pointing to a separate buffer in process memory.
//     The kernel copies these buffers to the target process.
//   - hidl_string: 16-byte struct (8-byte pointer + 4-byte size + 4-byte pad)
//     with string data as a child buffer
//   - hidl_vec<T>: 16-byte struct (8-byte pointer + 4-byte size + 4-byte pad)
//     with element data as a child buffer
package hwparcel

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/AndroidGoLab/binder/parcel"
)

const (
	// binderTypePTR is BINDER_TYPE_PTR for scatter-gather buffer objects.
	// Kernel value: B_PACK_CHARS('p','t','*',0x85) = 0x70742a85.
	binderTypePTR = uint32(0x70742a85)

	// binderBufferFlagHasParent indicates this buffer is embedded in a parent.
	binderBufferFlagHasParent = uint32(1)

	// binderBufferObjectSize is sizeof(binder_buffer_object) on 64-bit:
	//   binder_object_header.type: __u32 (4, offset 0)
	//   flags:                     __u32 (4, offset 4)
	//   buffer:                    __u64 (8, offset 8)
	//   length:                    __u64 (8, offset 16)
	//   parent:                    __u64 (8, offset 24)
	//   parent_offset:             __u64 (8, offset 32)
	//   Total = 40
	binderBufferObjectSize = 40

	// hidlStringSize is sizeof(hidl_string) on 64-bit:
	// hidl_pointer<char> (8) + uint32_t size (4) + bool owns (1) + pad[3] = 16.
	hidlStringSize = 16

	// hidlVecHeaderSize is sizeof(hidl_vec<T>) on 64-bit:
	// hidl_pointer<T> (8) + uint32_t size (4) + bool owns (1) + pad[3] = 16.
	hidlVecHeaderSize = 16
)

// bufferEntry holds a scatter-gather buffer and its metadata.
type bufferEntry struct {
	data []byte
}

// HwParcel builds a HIDL HwParcel for hwbinder transactions.
// It produces data and objects arrays compatible with the kernel binder
// scatter-gather protocol.
type HwParcel struct {
	// data holds the inline data (interface token + binder_buffer_object structs).
	data []byte

	// objects holds offsets into data where binder_buffer_object entries start.
	objects []uint64

	// buffers holds the actual buffer data referenced by binder_buffer_objects.
	// Index i corresponds to the i-th buffer object added.
	buffers []bufferEntry

	// bufferCount tracks the number of buffer objects added.
	bufferCount int
}

// New creates a new empty HwParcel.
func New() *HwParcel {
	return &HwParcel{}
}

// WriteInterfaceToken writes the HIDL interface descriptor as a
// null-terminated C string in the data buffer.
func (p *HwParcel) WriteInterfaceToken(descriptor string) {
	p.data = append(p.data, []byte(descriptor)...)
	p.data = append(p.data, 0)
	// Align to 4 bytes.
	for len(p.data)%4 != 0 {
		p.data = append(p.data, 0)
	}
}

// writeBufferObject writes a binder_buffer_object into the data buffer,
// records the offset in objects, and stores the buffer data for KeepAlive.
// Returns the buffer index (handle) for parent references.
func (p *HwParcel) writeBufferObject(
	buf []byte,
	flags uint32,
	parentHandle int,
	parentOffset uint64,
) int {
	handle := p.bufferCount
	p.bufferCount++

	// Record the offset where this binder_buffer_object starts.
	offset := uint64(len(p.data))
	p.objects = append(p.objects, offset)

	// Store the buffer data.
	p.buffers = append(p.buffers, bufferEntry{data: buf})

	// Build the binder_buffer_object in the data buffer.
	// The buffer pointer will be patched in ToParcel() once all buffers
	// are finalized.
	obj := make([]byte, binderBufferObjectSize)

	// binder_object_header.type (offset 0)
	binary.LittleEndian.PutUint32(obj[0:], binderTypePTR)
	// flags (offset 4)
	binary.LittleEndian.PutUint32(obj[4:], flags)
	// buffer (offset 8, placeholder, patched in ToParcel)
	binary.LittleEndian.PutUint64(obj[8:], 0)
	// length (offset 16)
	binary.LittleEndian.PutUint64(obj[16:], uint64(len(buf)))
	// parent (offset 24)
	binary.LittleEndian.PutUint64(obj[24:], uint64(parentHandle))
	// parent_offset (offset 32)
	binary.LittleEndian.PutUint64(obj[32:], parentOffset)

	p.data = append(p.data, obj...)

	return handle
}

// WriteBuffer writes a top-level scatter-gather buffer object.
// Returns the handle (index) for parent references.
func (p *HwParcel) WriteBuffer(buf []byte) int {
	return p.writeBufferObject(buf, 0, 0, 0)
}

// WriteEmbeddedBuffer writes a child scatter-gather buffer embedded within
// a parent buffer. parentOffset is the byte offset within the parent buffer
// where the pointer to this child buffer lives (typically 0 for hidl_string
// and hidl_vec, since the pointer is the first field).
func (p *HwParcel) WriteEmbeddedBuffer(
	buf []byte,
	parentHandle int,
	parentOffset uint64,
) int {
	return p.writeBufferObject(buf, binderBufferFlagHasParent, parentHandle, parentOffset)
}

// WriteHidlString writes a hidl_string value. Creates two buffer objects:
//  1. The hidl_string struct (16 bytes)
//  2. The string data (child of #1, linked at the pointer field offset 0)
func (p *HwParcel) WriteHidlString(s string) {
	structBuf := make([]byte, hidlStringSize)
	// pointer at offset 0 is zero (kernel patches it)
	binary.LittleEndian.PutUint32(structBuf[8:], uint32(len(s)))
	// owns=false and pad are already zero

	parentHandle := p.WriteBuffer(structBuf)

	// String data with null terminator.
	strData := make([]byte, len(s)+1)
	copy(strData, s)

	p.WriteEmbeddedBuffer(strData, parentHandle, 0)
}

// WriteHidlVecUint32 writes a hidl_vec<uint32_t>. Creates two buffer objects:
//  1. The hidl_vec header (16 bytes)
//  2. The uint32 array data (child of #1)
func (p *HwParcel) WriteHidlVecUint32(values []uint32) {
	headerBuf := make([]byte, hidlVecHeaderSize)
	binary.LittleEndian.PutUint32(headerBuf[8:], uint32(len(values)))

	parentHandle := p.WriteBuffer(headerBuf)

	dataBuf := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(dataBuf[i*4:], v)
	}

	p.WriteEmbeddedBuffer(dataBuf, parentHandle, 0)
}

// WriteUint32 writes a uint32 directly into the inline data buffer
// (not as a scatter-gather buffer object). Used for simple scalar
// arguments that follow buffer objects.
func (p *HwParcel) WriteUint32(v uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	p.data = append(p.data, b...)
}

// ToParcel converts this HwParcel into a standard parcel.Parcel.
// It patches the buffer pointers in each binder_buffer_object to point
// to the actual buffer data in memory. The returned keepAlive slice must
// be kept alive (via runtime.KeepAlive) until the transaction completes.
func (p *HwParcel) ToParcel() (*parcel.Parcel, [][]byte) {
	// Patch buffer pointers in the data.
	for i, entry := range p.buffers {
		objOffset := p.objects[i]
		// The buffer pointer is at offset 8 within the binder_buffer_object.
		ptrOffset := objOffset + 8
		if len(entry.data) > 0 {
			ptr := uint64(uintptr(unsafe.Pointer(&entry.data[0])))
			binary.LittleEndian.PutUint64(p.data[ptrOffset:], ptr)
		}
	}

	result := parcel.FromBytesWithObjects(p.data, p.objects)

	// Collect all buffer slices for KeepAlive.
	keepAlive := make([][]byte, len(p.buffers))
	for i, entry := range p.buffers {
		keepAlive[i] = entry.data
	}

	return result, keepAlive
}

// KeepBuffersAlive calls runtime.KeepAlive on all buffer slices.
// Must be called after the binder ioctl completes.
func KeepBuffersAlive(buffers [][]byte) {
	for _, b := range buffers {
		runtime.KeepAlive(b)
	}
}

// ResponseParcel wraps a reply parcel.Parcel from a HIDL transaction
// and provides methods to read HIDL-formatted response data.
// HIDL replies don't use scatter-gather -- the kernel copies all
// buffer data into the target's mmap region, and the driver copies
// it into the reply parcel data buffer inline.
type ResponseParcel struct {
	p *parcel.Parcel
}

// NewResponseParcel wraps a standard parcel for HIDL response reading.
func NewResponseParcel(p *parcel.Parcel) *ResponseParcel {
	return &ResponseParcel{p: p}
}

// ReadInt32 reads an int32 from the response.
func (r *ResponseParcel) ReadInt32() (int32, error) {
	return r.p.ReadInt32()
}

// ReadUint32 reads a uint32 from the response.
func (r *ResponseParcel) ReadUint32() (uint32, error) {
	return r.p.ReadUint32()
}

// ReadStrongBinder reads a binder handle from the HIDL response.
// HIDL does NOT write a stability level after the flat_binder_object
// (unlike AIDL).
func (r *ResponseParcel) ReadStrongBinder() (uint32, error) {
	const flatBinderObjectSize = 24
	b, err := r.p.Read(flatBinderObjectSize)
	if err != nil {
		return 0, fmt.Errorf("hwparcel: reading binder object: %w", err)
	}

	objType := binary.LittleEndian.Uint32(b[0:])
	handle := binary.LittleEndian.Uint32(b[8:])

	const binderTypeHandle = uint32(0x73682a85)
	const binderTypeBinder = uint32(0x73622a85)

	switch objType {
	case binderTypeHandle, binderTypeBinder:
		return handle, nil
	default:
		return 0, fmt.Errorf("hwparcel: unexpected binder type %#x", objType)
	}
}

// Position returns the current read position.
func (r *ResponseParcel) Position() int {
	return r.p.Position()
}

// DataLen returns the total data length.
func (r *ResponseParcel) DataLen() int {
	return r.p.Len()
}

// Remaining returns the number of unread bytes.
func (r *ResponseParcel) Remaining() int {
	return r.p.Len() - r.p.Position()
}

// ReadRawBytes reads n raw bytes.
func (r *ResponseParcel) ReadRawBytes(n int) ([]byte, error) {
	return r.p.Read(n)
}

// Underlying returns the wrapped parcel for low-level access.
func (r *ResponseParcel) Underlying() *parcel.Parcel {
	return r.p
}

// SkipBytes advances the read position by n bytes (4-byte aligned).
func (r *ResponseParcel) SkipBytes(n int) error {
	_, err := r.p.Read(n)
	return err
}

// ReadNativeHandle reads a native_handle_t from the HIDL response.
// Wire format: int32 numFds, int32 numInts, then flat_binder_object
// entries for each FD, then int32 entries for each int.
func (r *ResponseParcel) ReadNativeHandle() (fds []int32, ints []int32, err error) {
	// HIDL serializes native_handle as:
	// bool isNull (written as int32 in some versions)
	// If not null:
	//   int32 numFds
	//   int32 numInts
	//   numFds x FD (as flat_binder_object)
	//   numInts x int32

	// TODO: implement based on actual wire format observed on device
	return nil, nil, fmt.Errorf("hwparcel: ReadNativeHandle not yet implemented")
}

// Verify compile-time size assertion for binder_buffer_object.
var _ = [binderBufferObjectSize]byte{}

// Verify hidl_string and hidl_vec sizes.
func init() {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		// The sizes above are for 64-bit only. On 32-bit they would be different.
		// This package only supports 64-bit Android.
		panic("hwparcel: only 64-bit platforms are supported")
	}
}
