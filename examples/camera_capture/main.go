// Camera frame capture using gralloc-allocated buffers.
//
// This example connects to the camera via binder, allocates real gralloc
// buffers through IAllocator, captures frames, and verifies the pixel data.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/camera_capture ./examples/camera_capture/
//	adb push build/camera_capture /data/local/tmp/ && adb shell /data/local/tmp/camera_capture
package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"

	common "github.com/xaionaro-go/binder/android/hardware/common"
	"github.com/xaionaro-go/binder/android/hardware/graphics/allocator"
	gfxCommon "github.com/xaionaro-go/binder/android/hardware/graphics/common"

	fwkDevice "github.com/xaionaro-go/binder/android/frameworks/cameraservice/device"
	fwkService "github.com/xaionaro-go/binder/android/frameworks/cameraservice/service"
	"github.com/xaionaro-go/binder/binder"
	"github.com/xaionaro-go/binder/binder/versionaware"
	"github.com/xaionaro-go/binder/igbp"
	"github.com/xaionaro-go/binder/kernelbinder"
	"github.com/xaionaro-go/binder/parcel"
	"github.com/xaionaro-go/binder/servicemanager"

	"golang.org/x/sys/unix"
)

// --------------------------------------------------------------------
// Gralloc buffer allocation
// --------------------------------------------------------------------

// grallocBuffer holds a gralloc-allocated buffer with its NativeHandle.
type grallocBuffer struct {
	handle common.NativeHandle
	stride int32
	width  uint32
	height uint32
	format int32
	usage  uint64

	// mmapData holds a persistent read-only mmap of the dmabuf.
	mmapData []byte
}

// mmap creates a persistent read-only mmap of this buffer's dmabuf FD.
func (b *grallocBuffer) mmap() error {
	if len(b.handle.Fds) == 0 {
		return fmt.Errorf("no FDs in gralloc buffer")
	}
	fd := int(b.handle.Fds[0])
	bufSize := int(b.width) * int(b.height) * 3 / 2 // YCbCr_420_888
	data, err := unix.Mmap(fd, 0, bufSize, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("mmap fd=%d size=%d: %w", fd, bufSize, err)
	}
	b.mmapData = data
	return nil
}

// munmap releases the persistent mmap.
func (b *grallocBuffer) munmap() {
	if b.mmapData != nil {
		_ = unix.Munmap(b.mmapData)
		b.mmapData = nil
	}
}

// allocateGrallocBuffer allocates a gralloc buffer using the IAllocator
// HAL service. The returned buffer contains a dmabuf FD that can be
// mmap'd for CPU read access.
func allocateGrallocBuffer(
	ctx context.Context,
	sm *servicemanager.ServiceManager,
	width int32,
	height int32,
	format gfxCommon.PixelFormat,
	usage gfxCommon.BufferUsage,
) (*grallocBuffer, error) {
	svc, err := sm.GetService(ctx, "android.hardware.graphics.allocator.IAllocator/default")
	if err != nil {
		return nil, fmt.Errorf("get allocator service: %w", err)
	}

	proxy := allocator.NewAllocatorProxy(svc)

	desc := allocator.BufferDescriptorInfo{
		Name:              []byte("camera-buffer"),
		Width:             width,
		Height:            height,
		LayerCount:        1,
		Format:            format,
		Usage:             usage,
		ReservedSize:      0,
		AdditionalOptions: []gfxCommon.ExtendableType{},
	}

	result, err := proxy.Allocate2(ctx, desc, 1)
	if err != nil {
		return nil, fmt.Errorf("Allocate2: %w", err)
	}

	if len(result.Buffers) == 0 {
		return nil, fmt.Errorf("Allocate2 returned 0 buffers")
	}

	return &grallocBuffer{
		handle: result.Buffers[0],
		stride: result.Stride,
		width:  uint32(width),
		height: uint32(height),
		format: int32(format),
		usage:  uint64(usage),
	}, nil
}

// --------------------------------------------------------------------
// Camera device callbacks
// --------------------------------------------------------------------

type cameraCallback struct {
	mu             sync.Mutex
	framesReceived int
}

func (c *cameraCallback) OnCaptureStarted(
	_ context.Context,
	extras fwkDevice.CaptureResultExtras,
	ts int64,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.framesReceived++
	fmt.Printf("  >> OnCaptureStarted: requestId=%d timestamp=%d (total=%d)\n",
		extras.RequestId, ts, c.framesReceived)
	return nil
}

func (c *cameraCallback) OnDeviceError(
	_ context.Context,
	code fwkDevice.ErrorCode,
	extras fwkDevice.CaptureResultExtras,
) error {
	fmt.Printf("  >> OnDeviceError: code=%d requestId=%d\n", code, extras.RequestId)
	return nil
}

func (c *cameraCallback) OnDeviceIdle(_ context.Context) error {
	fmt.Println("  >> OnDeviceIdle")
	return nil
}

func (c *cameraCallback) OnPrepared(_ context.Context, streamId int32) error {
	fmt.Printf("  >> OnPrepared: stream %d\n", streamId)
	return nil
}

func (c *cameraCallback) OnRepeatingRequestError(
	_ context.Context,
	lastFrame int64,
	reqId int32,
) error {
	fmt.Printf("  >> OnRepeatingRequestError: frame=%d req=%d\n", lastFrame, reqId)
	return nil
}

func (c *cameraCallback) OnResultReceived(
	_ context.Context,
	meta fwkDevice.CaptureMetadataInfo,
	extras fwkDevice.CaptureResultExtras,
	_ []fwkDevice.PhysicalCaptureResultInfo,
) error {
	fmt.Printf("  >> OnResultReceived: requestId=%d frameNumber=%d tag=%d\n",
		extras.RequestId, extras.FrameNumber, meta.Tag)
	return nil
}

// --------------------------------------------------------------------
// IGraphicBufferProducer stub with gralloc buffers
// --------------------------------------------------------------------

// slotBuffer holds per-slot buffer state backed by a gralloc buffer.
type slotBuffer struct {
	gralloc *grallocBuffer
}

// graphicBufferProducerStub implements a minimal IGraphicBufferProducer
// that provides gralloc-allocated buffers to the camera HAL.
type graphicBufferProducerStub struct {
	width  uint32
	height uint32
	format int32

	grallocBufs [4]*grallocBuffer

	mu       sync.Mutex
	nextSlot int
	slots    [igbp.MaxBufferSlots]*slotBuffer
	queuedCh chan int // slot index of queued buffer
}

func newGraphicBufferProducerStub(
	width uint32,
	height uint32,
	grallocBufs [4]*grallocBuffer,
) *graphicBufferProducerStub {
	return &graphicBufferProducerStub{
		width:       width,
		height:      height,
		format:      int32(igbp.PixelFormatYCbCr420_888),
		grallocBufs: grallocBufs,
		queuedCh:    make(chan int, 16),
	}
}

func (g *graphicBufferProducerStub) Descriptor() string {
	return igbp.Descriptor
}

func (g *graphicBufferProducerStub) OnTransaction(
	_ context.Context,
	code binder.TransactionCode,
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	if _, err := data.ReadInterfaceToken(); err != nil {
		return nil, fmt.Errorf("IGBP: reading interface token: %w", err)
	}

	switch code {
	case igbp.RequestBuffer:
		return g.onRequestBuffer(data)
	case igbp.DequeueBuffer:
		return g.onDequeueBuffer(data)
	case igbp.QueueBuffer:
		return g.onQueueBuffer(data)
	case igbp.CancelBuffer:
		return g.onCancelBuffer(data)
	case igbp.Query:
		return g.onQuery(data)
	case igbp.Connect:
		return g.onConnect(data)
	case igbp.Disconnect:
		return g.onDisconnect(data)
	case igbp.SetMaxDequeuedBufCount,
		igbp.SetAsyncMode,
		igbp.AllowAllocation,
		igbp.SetGenerationNumber,
		igbp.SetDequeueTimeout,
		igbp.SetSharedBufferMode,
		igbp.SetAutoRefresh,
		igbp.SetLegacyBufferDrop,
		igbp.SetAutoPrerotation,
		igbp.DetachBuffer:
		return g.onSimpleOK()
	case igbp.GetConsumerName:
		return g.onGetConsumerName()
	case igbp.GetUniqueId:
		return g.onGetUniqueId()
	case igbp.GetConsumerUsage:
		return g.onGetConsumerUsage()
	case igbp.AllocateBuffers, igbp.GetFrameTimestamps:
		return nil, nil // void
	case igbp.GetLastQueuedBuffer:
		return g.onGetLastQueuedBuffer()
	default:
		reply := parcel.New()
		reply.WriteInt32(int32(igbp.StatusNoInit))
		return reply, nil
	}
}

func (g *graphicBufferProducerStub) onSimpleOK() (*parcel.Parcel, error) {
	reply := parcel.New()
	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

// writeGrallocGraphicBufferToParcel writes a flattened GraphicBuffer backed
// by a gralloc NativeHandle. The wire format follows GraphicBuffer::flatten():
//
//	int32(flattenedSize) + int32(fdCount) + raw[flattenedSize] + fd objects
func writeGrallocGraphicBufferToParcel(
	p *parcel.Parcel,
	buf *grallocBuffer,
	bufID uint64,
) {
	numFds := int32(len(buf.handle.Fds))
	numInts := int32(len(buf.handle.Ints))

	// Flattened size: 13 int32s (header) + numInts int32s.
	flattenedSize := (13 + numInts) * 4

	p.WriteInt32(flattenedSize)
	p.WriteInt32(numFds)

	raw := make([]byte, flattenedSize)
	binary.LittleEndian.PutUint32(raw[0:], uint32(igbp.GraphicBufferMagicGB01))
	binary.LittleEndian.PutUint32(raw[4:], buf.width)
	binary.LittleEndian.PutUint32(raw[8:], buf.height)
	binary.LittleEndian.PutUint32(raw[12:], uint32(buf.stride))
	binary.LittleEndian.PutUint32(raw[16:], uint32(buf.format))
	binary.LittleEndian.PutUint32(raw[20:], 1)                        // layerCount
	binary.LittleEndian.PutUint32(raw[24:], uint32(buf.usage))        // usage low
	binary.LittleEndian.PutUint32(raw[28:], uint32(bufID>>32))        // id high
	binary.LittleEndian.PutUint32(raw[32:], uint32(bufID&0xFFFFFFFF)) // id low
	binary.LittleEndian.PutUint32(raw[36:], 0)                        // generationNumber
	binary.LittleEndian.PutUint32(raw[40:], uint32(numFds))           // numFds
	binary.LittleEndian.PutUint32(raw[44:], uint32(numInts))          // numInts
	binary.LittleEndian.PutUint32(raw[48:], uint32(buf.usage>>32))    // usage high

	// Append native_handle ints after the 13-word header.
	for i, v := range buf.handle.Ints {
		binary.LittleEndian.PutUint32(raw[52+i*4:], uint32(v))
	}

	p.WriteRawBytes(raw)

	// Write each FD as a flat_binder_object.
	for _, fd := range buf.handle.Fds {
		p.WriteFileDescriptor(fd)
	}
}

func (g *graphicBufferProducerStub) onRequestBuffer(
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	slot, _ := data.ReadInt32()

	g.mu.Lock()
	buf := g.slots[slot]
	g.mu.Unlock()

	reply := parcel.New()
	if buf == nil {
		reply.WriteInt32(0) // nonNull=0
		reply.WriteInt32(int32(igbp.StatusOK))
		return reply, nil
	}

	reply.WriteInt32(1) // nonNull=1

	bufID := uint64(0xCAFE0000) | uint64(slot)
	writeGrallocGraphicBufferToParcel(reply, buf.gralloc, bufID)

	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

func (g *graphicBufferProducerStub) onDequeueBuffer(
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	_, _ = data.ReadUint32()  // w
	_, _ = data.ReadUint32()  // h
	_, _ = data.ReadInt32()   // format
	_, _ = data.ReadUint64()  // usage
	getTimestamps, _ := data.ReadBool()

	g.mu.Lock()
	slot := g.nextSlot
	g.nextSlot = (g.nextSlot + 1) % 4

	needsRealloc := false
	if g.slots[slot] == nil {
		needsRealloc = true
		g.slots[slot] = &slotBuffer{
			gralloc: g.grallocBufs[slot],
		}
	}
	g.mu.Unlock()

	reply := parcel.New()
	reply.WriteInt32(int32(slot))

	// Fence as Flattenable: flattenedSize=4, fdCount=0, numFds=0.
	reply.WriteInt32(4)
	reply.WriteInt32(0)
	reply.WriteUint32(0)

	// bufferAge.
	reply.WriteUint64(0)

	if getTimestamps {
		writeEmptyFrameEventHistoryDelta(reply)
	}

	if needsRealloc {
		reply.WriteInt32(int32(igbp.BufferNeedsRealloc))
	} else {
		reply.WriteInt32(int32(igbp.StatusOK))
	}
	return reply, nil
}

func writeEmptyFrameEventHistoryDelta(reply *parcel.Parcel) {
	reply.WriteInt32(28) // flattenedSize
	reply.WriteInt32(0)  // fdCount
	reply.WriteInt64(0)  // compositor deadline
	reply.WriteInt64(0)  // compositor interval
	reply.WriteInt64(0)  // compositor presentLatency
	reply.WriteInt32(0)  // delta count
}

// queueBufferOutputPayload is a pre-computed 61-byte QueueBufferOutput flattenable payload.
var queueBufferOutputPayload = func() []byte {
	const flatSize = 61
	buf := make([]byte, flatSize)
	off := 0
	binary.LittleEndian.PutUint32(buf[off:], 0) // width
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], 0) // height
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], 0) // transformHint
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], 0) // numPendingBuffers
	off += 4
	binary.LittleEndian.PutUint64(buf[off:], 1) // nextFrameNumber
	off += 8
	buf[off] = 0 // bufferReplaced
	off += 1
	binary.LittleEndian.PutUint32(buf[off:], 64) // maxBufferCount
	off += 4
	off += 24                                   // compositorTiming zeros
	binary.LittleEndian.PutUint32(buf[off:], 0) // deltaCount
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], 0) // result
	_ = off
	return buf
}()

func writeQueueBufferOutput(reply *parcel.Parcel) {
	reply.WriteInt32(int32(len(queueBufferOutputPayload)))
	reply.WriteInt32(0) // fdCount
	reply.WriteRawBytes(queueBufferOutputPayload)
}

func (g *graphicBufferProducerStub) onQueueBuffer(
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	slot, _ := data.ReadInt32()

	select {
	case g.queuedCh <- int(slot):
	default:
	}

	reply := parcel.New()
	writeQueueBufferOutput(reply)
	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

func (g *graphicBufferProducerStub) onCancelBuffer(
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	_, _ = data.ReadInt32()
	reply := parcel.New()
	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

func (g *graphicBufferProducerStub) onQuery(
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	rawWhat, _ := data.ReadInt32()
	what := igbp.NativeWindowQuery(rawWhat)

	var value int32
	switch what {
	case igbp.NativeWindowWidth:
		value = int32(g.width)
	case igbp.NativeWindowHeight:
		value = int32(g.height)
	case igbp.NativeWindowFormat:
		value = g.format
	case igbp.NativeWindowMinUndequeued:
		value = 1
	case igbp.NativeWindowQueuesToComposer:
		value = 0
	case igbp.NativeWindowConcreteType:
		value = int32(igbp.NativeWindowSurface)
	case igbp.NativeWindowDefaultWidth:
		value = int32(g.width)
	case igbp.NativeWindowDefaultHeight:
		value = int32(g.height)
	case igbp.NativeWindowTransformHint:
		value = 0
	case igbp.NativeWindowConsumerRunning:
		value = 0
	case igbp.NativeWindowConsumerUsageBits:
		value = 0
	case igbp.NativeWindowStickyTransform:
		value = 0
	case igbp.NativeWindowDefaultDataspace:
		value = 0
	case igbp.NativeWindowBufferAge:
		value = 0
	case igbp.NativeWindowMaxBufferCount:
		value = 64
	}

	reply := parcel.New()
	reply.WriteInt32(value)
	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

func (g *graphicBufferProducerStub) onConnect(
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	hasListener, _ := data.ReadInt32()
	if hasListener == 1 {
		_, _ = data.ReadStrongBinder()
	}
	_, _ = data.ReadInt32() // api
	_, _ = data.ReadInt32() // producerControlled

	reply := parcel.New()
	writeQueueBufferOutput(reply)
	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

func (g *graphicBufferProducerStub) onDisconnect(
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	_, _ = data.ReadInt32()
	reply := parcel.New()
	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

func (g *graphicBufferProducerStub) onGetConsumerName() (*parcel.Parcel, error) {
	reply := parcel.New()
	reply.WriteString16("GoCamera")
	return reply, nil
}

func (g *graphicBufferProducerStub) onGetUniqueId() (*parcel.Parcel, error) {
	reply := parcel.New()
	reply.WriteUint64(0x12345678)
	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

func (g *graphicBufferProducerStub) onGetConsumerUsage() (*parcel.Parcel, error) {
	reply := parcel.New()
	reply.WriteUint64(0)
	reply.WriteInt32(int32(igbp.StatusOK))
	return reply, nil
}

func (g *graphicBufferProducerStub) onGetLastQueuedBuffer() (*parcel.Parcel, error) {
	reply := parcel.New()
	reply.WriteInt32(int32(igbp.StatusNoInit))
	return reply, nil
}

// --------------------------------------------------------------------
// Raw parcel helpers for camera operations
// --------------------------------------------------------------------

func createStreamWithSurface(
	ctx context.Context,
	deviceUser fwkDevice.ICameraDeviceUser,
	transport binder.Transport,
	igbpStub *binder.StubBinder,
	width int32,
	height int32,
) (int32, error) {
	data := parcel.New()
	data.WriteInterfaceToken(fwkDevice.DescriptorICameraDeviceUser)

	data.WriteInt32(1) // non-null OutputConfiguration

	headerPos := parcel.WriteParcelableHeader(data)

	data.WriteInt32(0)  // windowHandles: empty array
	data.WriteInt32(0)  // rotation: R0
	data.WriteInt32(-1) // windowGroupId: NONE
	data.WriteString16("")
	data.WriteInt32(width)
	data.WriteInt32(height)
	data.WriteBool(false) // isDeferred

	// surfaces: array of 1.
	data.WriteInt32(1)
	data.WriteInt32(1) // non-null

	// view::Surface::writeToParcel
	data.WriteString16("GoCamera")
	data.WriteInt32(0)           // isSingleBuffered
	data.WriteUint32(0x62717565) // USE_BUFFER_QUEUE
	binder.WriteBinderToParcel(ctx, data, igbpStub, transport)
	data.WriteNullStrongBinder() // surfaceControlHandle: null

	parcel.WriteParcelableFooter(data, headerPos)

	reply, err := deviceUser.AsBinder().Transact(
		ctx,
		fwkDevice.TransactionICameraDeviceUserCreateStream,
		0,
		data,
	)
	if err != nil {
		return 0, fmt.Errorf("transaction: %w", err)
	}
	defer reply.Recycle()

	if err = binder.ReadStatus(reply); err != nil {
		return 0, fmt.Errorf("status: %w", err)
	}

	streamId, err := reply.ReadInt32()
	if err != nil {
		return 0, fmt.Errorf("readStreamId: %w", err)
	}
	return streamId, nil
}

func createDefaultRequestRaw(
	ctx context.Context,
	deviceUser fwkDevice.ICameraDeviceUser,
	templateId fwkDevice.TemplateId,
) ([]byte, error) {
	data := parcel.New()
	data.WriteInterfaceToken(fwkDevice.DescriptorICameraDeviceUser)
	data.WriteInt32(int32(templateId))

	reply, err := deviceUser.AsBinder().Transact(
		ctx,
		fwkDevice.TransactionICameraDeviceUserCreateDefaultRequest,
		0,
		data,
	)
	if err != nil {
		return nil, fmt.Errorf("transaction: %w", err)
	}
	defer reply.Recycle()

	if err = binder.ReadStatus(reply); err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	nullInd, err := reply.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("null indicator: %w", err)
	}
	if nullInd == 0 {
		return nil, fmt.Errorf("null metadata")
	}

	endPos, err := parcel.ReadParcelableHeader(reply)
	if err != nil {
		return nil, fmt.Errorf("parcelable header: %w", err)
	}

	metadataBytes, err := reply.ReadByteArray()
	if err != nil {
		return nil, fmt.Errorf("ReadByteArray: %w", err)
	}

	parcel.SkipToParcelableEnd(reply, endPos)
	return metadataBytes, nil
}

func submitRequestRaw(
	ctx context.Context,
	deviceUser fwkDevice.ICameraDeviceUser,
	captureReq fwkDevice.CaptureRequest,
	isRepeating bool,
) (fwkDevice.SubmitInfo, error) {
	var result fwkDevice.SubmitInfo

	data := parcel.New()
	data.WriteInterfaceToken(fwkDevice.DescriptorICameraDeviceUser)
	data.WriteInt32(1) // requestList: array of 1
	data.WriteInt32(1) // non-null indicator
	if err := captureReq.MarshalParcel(data); err != nil {
		return result, fmt.Errorf("marshal: %w", err)
	}
	data.WriteBool(isRepeating)

	reply, err := deviceUser.AsBinder().Transact(
		ctx,
		fwkDevice.TransactionICameraDeviceUserSubmitRequestList,
		0,
		data,
	)
	if err != nil {
		return result, fmt.Errorf("transaction: %w", err)
	}
	defer reply.Recycle()

	if err = binder.ReadStatus(reply); err != nil {
		return result, fmt.Errorf("status: %w", err)
	}

	nullInd, err := reply.ReadInt32()
	if err != nil {
		return result, fmt.Errorf("null indicator: %w", err)
	}
	if nullInd != 0 {
		if err = result.UnmarshalParcel(reply); err != nil {
			return result, fmt.Errorf("unmarshal SubmitInfo: %w", err)
		}
	}

	return result, nil
}

// --------------------------------------------------------------------
// Main flow
// --------------------------------------------------------------------

func run(ctx context.Context) error {
	const (
		cameraID = "0"
		width    = 640
		height   = 480
	)

	drv, err := kernelbinder.Open(ctx, binder.WithMapSize(4*1024*1024))
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer drv.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		return fmt.Errorf("transport: %w", err)
	}
	sm := servicemanager.New(transport)

	// Step 0: Pre-allocate gralloc buffers.
	fmt.Println("=== Step 0: Allocate gralloc buffers ===")
	var grallocBufs [4]*grallocBuffer
	for i := range grallocBufs {
		buf, allocErr := allocateGrallocBuffer(
			ctx,
			sm,
			width,
			height,
			gfxCommon.PixelFormatYcbcr420888,
			gfxCommon.BufferUsageCpuReadOften|gfxCommon.BufferUsageCameraOutput,
		)
		if allocErr != nil {
			return fmt.Errorf("allocating gralloc buffer %d: %w", i, allocErr)
		}
		if mmapErr := buf.mmap(); mmapErr != nil {
			return fmt.Errorf("mmap gralloc buffer %d: %w", i, mmapErr)
		}
		grallocBufs[i] = buf
		fmt.Printf("  Buffer %d: stride=%d fds=%v ints_count=%d\n",
			i, buf.stride, buf.handle.Fds, len(buf.handle.Ints))
	}
	defer func() {
		for _, buf := range grallocBufs {
			if buf != nil {
				buf.munmap()
			}
		}
	}()

	// Step 1: Connect to camera service.
	fmt.Println("\n=== Step 1: Connect to camera ===")
	svc, err := sm.GetService(ctx, "android.frameworks.cameraservice.service.ICameraService/default")
	if err != nil {
		return fmt.Errorf("getting camera service: %w", err)
	}

	proxy := fwkService.NewCameraServiceProxy(svc)
	cb := &cameraCallback{}
	stub := fwkDevice.NewCameraDeviceCallbackStub(cb)

	stubBinder := stub.AsBinder().(*binder.StubBinder)
	stubBinder.RegisterWithTransport(ctx, transport)
	time.Sleep(100 * time.Millisecond)

	deviceUser, err := proxy.ConnectDevice(ctx, stub, cameraID)
	if err != nil {
		return fmt.Errorf("ConnectDevice: %w", err)
	}
	fmt.Println("ConnectDevice OK")

	defer func() {
		if disconnectErr := deviceUser.Disconnect(ctx); disconnectErr != nil {
			fmt.Printf("Disconnect: %v\n", disconnectErr)
		}
	}()

	// Step 2: Configure stream.
	fmt.Println("\n=== Step 2: Configure stream ===")
	if err = deviceUser.BeginConfigure(ctx); err != nil {
		return fmt.Errorf("BeginConfigure: %w", err)
	}

	metadataBytes, err := createDefaultRequestRaw(ctx, deviceUser, fwkDevice.TemplateIdPREVIEW)
	if err != nil {
		return fmt.Errorf("CreateDefaultRequest: %w", err)
	}

	igbpStub := newGraphicBufferProducerStub(width, height, grallocBufs)
	igbpStubBinder := binder.NewStubBinder(igbpStub)
	igbpStubBinder.RegisterWithTransport(ctx, transport)

	streamId, err := createStreamWithSurface(ctx, deviceUser, transport, igbpStubBinder, width, height)
	if err != nil {
		return fmt.Errorf("CreateStream: %w", err)
	}
	fmt.Printf("CreateStream OK: streamId=%d\n", streamId)

	if err = deviceUser.EndConfigure(
		ctx,
		fwkDevice.StreamConfigurationModeNormalMode,
		fwkDevice.CameraMetadata{Metadata: []byte{}},
		0,
	); err != nil {
		return fmt.Errorf("EndConfigure: %w", err)
	}
	fmt.Println("EndConfigure OK")

	// Step 3: Submit capture request.
	fmt.Println("\n=== Step 3: Capture frames ===")
	captureReq := fwkDevice.CaptureRequest{
		PhysicalCameraSettings: []fwkDevice.PhysicalCameraSettings{
			{
				Id: cameraID,
				Settings: fwkDevice.CaptureMetadataInfo{
					Tag:      fwkDevice.CaptureMetadataInfoTagMetadata,
					Metadata: fwkDevice.CameraMetadata{Metadata: metadataBytes},
				},
			},
		},
		StreamAndWindowIds: []fwkDevice.StreamAndWindowId{
			{StreamId: streamId, WindowId: 0},
		},
	}

	submitInfo, err := submitRequestRaw(ctx, deviceUser, captureReq, true)
	if err != nil {
		fmt.Printf("SubmitRequestList (repeating) failed: %v; trying single shot\n", err)
		submitInfo, err = submitRequestRaw(ctx, deviceUser, captureReq, false)
		if err != nil {
			return fmt.Errorf("SubmitRequestList: %w", err)
		}
	}
	fmt.Printf("SubmitRequestList OK: requestId=%d\n", submitInfo.RequestId)

	// Step 4: Wait for frames and verify pixel data.
	fmt.Println("\nWaiting for frames (up to 10 seconds)...")
	framesWithData := 0
	deadline := time.After(10 * time.Second)
	for framesWithData < 3 {
		select {
		case <-deadline:
			if framesWithData == 0 {
				return fmt.Errorf("timeout: no frames with data received")
			}
			goto done
		case slot := <-igbpStub.queuedCh:
			igbpStub.mu.Lock()
			buf := igbpStub.slots[slot]
			igbpStub.mu.Unlock()

			if buf == nil || buf.gralloc.mmapData == nil {
				continue
			}

			frameData := buf.gralloc.mmapData
			nonZero := 0
			for _, b := range frameData {
				if b != 0 {
					nonZero++
				}
			}

			totalPixels := len(frameData)
			nonZeroPct := float64(nonZero) * 100 / float64(totalPixels)
			fmt.Printf("  Frame from slot %d: %d/%d bytes non-zero (%.1f%%)\n",
				slot, nonZero, totalPixels, nonZeroPct)

			if nonZero > 0 {
				framesWithData++
			}

			// Save first frame as raw file.
			if framesWithData == 1 {
				outputPath := "/data/local/tmp/camera_frame.yuv"
				if writeErr := os.WriteFile(outputPath, frameData, 0666); writeErr != nil {
					fmt.Printf("  Warning: could not save frame: %v\n", writeErr)
				} else {
					fmt.Printf("  Saved first frame to %s (%d bytes)\n", outputPath, len(frameData))
				}
			}
		}
	}

done:
	fmt.Printf("\nSuccess! Captured %d frames with non-zero pixel data.\n", framesWithData)
	return nil
}

func main() {
	ctx := context.Background()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
}
