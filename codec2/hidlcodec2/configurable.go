package hidlcodec2

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
	configurableFQName = "android.hardware.media.c2@1.0::IConfigurable"
)

// Configurable is a HIDL client for IConfigurable.
type Configurable struct {
	transport binder.Transport
	handle    uint32
}

// GetName calls IConfigurable::getName() and returns the component name.
func (cfg *Configurable) GetName(
	ctx context.Context,
) (_ string, _err error) {
	logger.Tracef(ctx, "hidlcodec2.Configurable.GetName")
	defer func() { logger.Tracef(ctx, "/hidlcodec2.Configurable.GetName: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(configurableFQName)

	reply, err := hwservicemanager.TransactHidl(
		ctx, cfg.transport, cfg.handle, transactionGetName, hp,
	)
	if err != nil {
		return "", fmt.Errorf("getName transaction: %w", err)
	}
	defer reply.Recycle()

	return parseGetNameResponse(reply.Data())
}

// parseGetNameResponse parses the reply from IConfigurable::getName().
//
// Wire format:
//
//	[0:4]  int32 HIDL status
//	[4:44] binder_buffer_object #0: hidl_string header (16 bytes)
//	[44:84] binder_buffer_object #1: string data (child of #0)
func parseGetNameResponse(data []byte) (string, error) {
	if len(data) < 4 {
		return "", fmt.Errorf("getName response too short: %d bytes", len(data))
	}

	hidlStatus := int32(binary.LittleEndian.Uint32(data[0:]))
	if hidlStatus != 0 {
		return "", fmt.Errorf("getName HIDL status error: %d", hidlStatus)
	}

	// Scan for PTR buffer objects after the status.
	pos := 4
	var buffers []resolvedBuffer
	for pos+binderBufferObjectSize <= len(data) {
		objType := binary.LittleEndian.Uint32(data[pos:])
		if objType != binderTypePTR {
			break
		}
		bufPtr := binary.LittleEndian.Uint64(data[pos+8:])
		bufLen := binary.LittleEndian.Uint64(data[pos+16:])
		buffers = append(buffers, resolvedBuffer{ptr: bufPtr, len: bufLen})
		pos += binderBufferObjectSize
	}

	// The second buffer (index 1) contains the string data.
	if len(buffers) < 2 {
		return "", fmt.Errorf("getName: expected 2 buffer objects, got %d", len(buffers))
	}

	s, err := readStringBuffer(data, buffers[1])
	if err != nil {
		return "", fmt.Errorf("getName: reading string: %w", err)
	}

	return s, nil
}

// Config calls IConfigurable::config() with the given C2 parameter blob.
//
// inParams is the concatenation of C2Param structs (each prefixed with
// uint32 size + uint32 index).
//
// Returns the C2 status and the output params blob.
func (cfg *Configurable) Config(
	ctx context.Context,
	inParams []byte,
	mayBlock bool,
) (_ Status, _ []byte, _err error) {
	logger.Tracef(ctx, "hidlcodec2.Configurable.Config(len=%d, mayBlock=%v)", len(inParams), mayBlock)
	defer func() { logger.Tracef(ctx, "/hidlcodec2.Configurable.Config: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(configurableFQName)

	// Arg 1: Params inParams (hidl_vec<uint8_t>).
	hp.WriteHidlVecBytes(inParams)

	// Arg 2: bool mayBlock.
	hp.WriteBool(mayBlock)

	reply, err := hwservicemanager.TransactHidl(
		ctx, cfg.transport, cfg.handle, transactionConfig, hp,
	)
	if err != nil {
		return StatusCorrupted, nil, fmt.Errorf("config transaction: %w", err)
	}
	defer reply.Recycle()

	return parseConfigResponse(reply.Data())
}

// parseConfigResponse parses the reply from IConfigurable::config().
//
// Wire format:
//
//	[0:4]   int32 HIDL status
//	[4:8]   int32 C2 Status
//	... binder_buffer_objects for vec<SettingResult> failures ...
//	... binder_buffer_objects for Params outParams ...
//
// We only extract the status and skip the failures/outParams for simplicity.
func parseConfigResponse(data []byte) (Status, []byte, error) {
	if len(data) < 8 {
		return StatusCorrupted, nil, fmt.Errorf("config response too short: %d bytes", len(data))
	}

	hidlStatus := int32(binary.LittleEndian.Uint32(data[0:]))
	if hidlStatus != 0 {
		return StatusCorrupted, nil, fmt.Errorf("config HIDL status error: %d", hidlStatus)
	}

	c2Status := Status(int32(binary.LittleEndian.Uint32(data[4:])))
	return c2Status, nil, nil
}

// GetId calls IConfigurable::getId().
func (cfg *Configurable) GetId(
	ctx context.Context,
) (_ uint32, _err error) {
	logger.Tracef(ctx, "hidlcodec2.Configurable.GetId")
	defer func() { logger.Tracef(ctx, "/hidlcodec2.Configurable.GetId: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(configurableFQName)

	reply, err := hwservicemanager.TransactHidl(
		ctx, cfg.transport, cfg.handle, transactionGetId, hp,
	)
	if err != nil {
		return 0, fmt.Errorf("getId transaction: %w", err)
	}
	defer reply.Recycle()

	data := reply.Data()
	if len(data) < 8 {
		return 0, fmt.Errorf("getId response too short: %d bytes", len(data))
	}

	hidlStatus := int32(binary.LittleEndian.Uint32(data[0:]))
	if hidlStatus != 0 {
		return 0, fmt.Errorf("getId HIDL status error: %d", hidlStatus)
	}

	id := binary.LittleEndian.Uint32(data[4:])
	return id, nil
}
