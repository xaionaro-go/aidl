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
	// storeFQName12 is the HIDL fully-qualified name for the 1.2 store.
	storeFQName12 = "android.hardware.media.c2@1.2::IComponentStore"

	// storeFQName10 is the HIDL fully-qualified name for the 1.0 store.
	storeFQName10 = "android.hardware.media.c2@1.0::IComponentStore"

	// storeInstance is the service instance name.
	storeInstance = "software"
)

// ComponentStore is a HIDL client for IComponentStore on hwbinder.
type ComponentStore struct {
	transport binder.Transport
	handle    uint32
	fqName    string
}

// GetComponentStore looks up the Codec2 IComponentStore service via
// hwservicemanager and returns a client. It tries @1.2 first, then
// falls back to @1.0.
func GetComponentStore(
	ctx context.Context,
	transport binder.Transport,
) (_ *ComponentStore, _err error) {
	logger.Tracef(ctx, "hidlcodec2.GetComponentStore")
	defer func() { logger.Tracef(ctx, "/hidlcodec2.GetComponentStore: %v", _err) }()

	sm := hwservicemanager.New(transport)

	// Try 1.2 first (goldfish emulator typically runs this version).
	handle, err := sm.GetService(ctx, storeFQName12, storeInstance)
	if err == nil {
		logger.Debugf(ctx, "connected to %s/%s handle=%d", storeFQName12, storeInstance, handle)
		return &ComponentStore{
			transport: transport,
			handle:    handle,
			fqName:    storeFQName12,
		}, nil
	}
	logger.Debugf(ctx, "%s/%s not found: %v; trying 1.0", storeFQName12, storeInstance, err)

	// Fall back to 1.0.
	handle, err = sm.GetService(ctx, storeFQName10, storeInstance)
	if err != nil {
		return nil, fmt.Errorf("get IComponentStore service: %w", err)
	}

	logger.Debugf(ctx, "connected to %s/%s handle=%d", storeFQName10, storeInstance, handle)
	return &ComponentStore{
		transport: transport,
		handle:    handle,
		fqName:    storeFQName10,
	}, nil
}

// Handle returns the binder handle for this store.
func (s *ComponentStore) Handle() uint32 {
	return s.handle
}

// ListComponents calls IComponentStore::listComponents() and returns the
// available codec traits.
func (s *ComponentStore) ListComponents(
	ctx context.Context,
) (_ []ComponentTraits, _err error) {
	logger.Tracef(ctx, "hidlcodec2.ListComponents")
	defer func() { logger.Tracef(ctx, "/hidlcodec2.ListComponents: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(storeFQName10)

	reply, err := hwservicemanager.TransactHidl(
		ctx, s.transport, s.handle, transactionListComponents, hp,
	)
	if err != nil {
		return nil, fmt.Errorf("listComponents transaction: %w", err)
	}
	defer reply.Recycle()

	return parseListComponentsResponse(reply.Data())
}

// parseListComponentsResponse parses the raw HIDL reply for listComponents().
//
// HIDL reply wire format (after scatter-gather resolution by kernel):
//
//	[0:4]   int32 HIDL status (0 = OK)
//	[4:8]   int32 C2 Status
//	[8:48]  binder_buffer_object #0: hidl_vec<ComponentTraits> header (16 bytes buffer)
//	[48:88] binder_buffer_object #1: ComponentTraits array data (child of #0)
//	...additional embedded buffer objects for strings/vecs within each trait...
//
// After the kernel resolves scatter-gather, all buffer data is inlined
// after the binder_buffer_object entries. The buffer pointers become
// offsets into the reply data.
func parseListComponentsResponse(data []byte) ([]ComponentTraits, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	hidlStatus := int32(binary.LittleEndian.Uint32(data[0:]))
	if hidlStatus != 0 {
		return nil, fmt.Errorf("HIDL status error: %d", hidlStatus)
	}

	c2Status := Status(int32(binary.LittleEndian.Uint32(data[4:])))
	if err := c2Status.Err(); err != nil {
		return nil, fmt.Errorf("listComponents: %w", err)
	}

	// Scan for binder_buffer_object entries to find the resolved buffer
	// data. The first PTR object is the hidl_vec header (16 bytes),
	// containing count at offset 8. The second PTR is the array of
	// ComponentTraits structs.
	pos := 8
	var buffers []resolvedBuffer
	for pos+binderBufferObjectSize <= len(data) {
		objType := binary.LittleEndian.Uint32(data[pos:])
		if objType != binderTypePTR {
			break
		}
		bufPtr := binary.LittleEndian.Uint64(data[pos+8:])
		bufLen := binary.LittleEndian.Uint64(data[pos+16:])
		buffers = append(buffers, resolvedBuffer{
			ptr: bufPtr,
			len: bufLen,
		})
		pos += binderBufferObjectSize
	}

	if len(buffers) < 1 {
		return nil, fmt.Errorf("no buffer objects in listComponents reply")
	}

	// The first buffer is the hidl_vec header: ptr(8) + size(4) + owns(4).
	vecHeader := buffers[0]
	if vecHeader.len < 16 {
		return nil, fmt.Errorf("vec header too short: %d", vecHeader.len)
	}
	headerData, err := readResolvedBuffer(data, vecHeader)
	if err != nil {
		return nil, fmt.Errorf("reading vec header: %w", err)
	}
	count := binary.LittleEndian.Uint32(headerData[8:])

	if count == 0 {
		return nil, nil
	}

	// The remaining buffer objects contain the flattened ComponentTraits
	// data. Each ComponentTraits has nested hidl_strings and hidl_vecs,
	// each serialized as its own buffer object pair (header + data).
	//
	// ComponentTraits layout (each field is a buffer object pair):
	//   hidl_string name      (2 buffers: header + string data)
	//   uint32      domain    (inline in the traits array buffer)
	//   uint32      kind      (inline)
	//   uint32      rank      (inline)
	//   hidl_string mediaType (2 buffers)
	//   hidl_vec<hidl_string> aliases (2+ buffers)
	//
	// The second buffer object contains the traits array: each element is
	// a fixed-size struct with embedded hidl_string/hidl_vec headers whose
	// data pointers are resolved into subsequent buffer objects.
	if len(buffers) < 2 {
		return nil, fmt.Errorf("missing traits array buffer object")
	}

	// Each ComponentTraits inline struct in the array:
	//   [0:16]  hidl_string name   (16 bytes)
	//   [16:20] uint32 domain
	//   [20:24] uint32 kind
	//   [24:28] uint32 rank
	//   [28:32] padding
	//   [32:48] hidl_string mediaType (16 bytes)
	//   [48:64] hidl_vec<hidl_string> aliases (16 bytes)
	// Total: 64 bytes per trait.
	const traitSize = 64

	traitsArrayBuf := buffers[1]
	traitsData, err := readResolvedBuffer(data, traitsArrayBuf)
	if err != nil {
		return nil, fmt.Errorf("reading traits array: %w", err)
	}

	if uint32(len(traitsData)) < count*traitSize {
		return nil, fmt.Errorf("traits array too short: have %d, need %d",
			len(traitsData), count*traitSize)
	}

	// Buffer objects after index 1 contain the string/vec data for each
	// trait's fields. The order is:
	//   For each trait: name_data, mediaType_data, aliases_header_data, [alias_string_data...]
	// That is 2 string buffers + aliases per trait, but the exact count
	// depends on how many aliases each trait has.
	//
	// We parse the string data from subsequent buffer objects in order.
	bufIdx := 2

	traits := make([]ComponentTraits, count)
	for i := uint32(0); i < count; i++ {
		offset := i * traitSize
		t := traitsData[offset : offset+traitSize]

		traits[i].Domain = Domain(binary.LittleEndian.Uint32(t[16:]))
		traits[i].Kind = Kind(binary.LittleEndian.Uint32(t[20:]))
		traits[i].Rank = binary.LittleEndian.Uint32(t[24:])

		// Read name string from the next buffer object.
		if bufIdx < len(buffers) {
			s, err := readStringBuffer(data, buffers[bufIdx])
			if err == nil {
				traits[i].Name = s
			}
			bufIdx++
		}

		// Read mediaType string from the next buffer object.
		if bufIdx < len(buffers) {
			s, err := readStringBuffer(data, buffers[bufIdx])
			if err == nil {
				traits[i].MediaType = s
			}
			bufIdx++
		}

		// Read aliases vec data.
		if bufIdx < len(buffers) {
			aliasesHdr := buffers[bufIdx]
			bufIdx++
			hdrData, hdrErr := readResolvedBuffer(data, aliasesHdr)
			aliasCount := uint32(0)
			if hdrErr == nil && len(hdrData) >= 12 {
				aliasCount = binary.LittleEndian.Uint32(hdrData[8:])
			}
			for j := uint32(0); j < aliasCount && bufIdx < len(buffers); j++ {
				// Each alias is a hidl_string: header buffer + data buffer.
				// The header buffer is embedded in the aliases vec data,
				// so only the data buffer appears here.
				s, sErr := readStringBuffer(data, buffers[bufIdx])
				if sErr == nil {
					traits[i].Aliases = append(traits[i].Aliases, s)
				}
				bufIdx++
			}
		}
	}

	return traits, nil
}

// resolvedBuffer describes a kernel-resolved scatter-gather buffer.
type resolvedBuffer struct {
	ptr uint64
	len uint64
}

// readResolvedBuffer reads the buffer data from the reply data at the
// resolved pointer offset. After scatter-gather resolution, the buffer
// pointer is an offset into the reply data.
func readResolvedBuffer(data []byte, buf resolvedBuffer) ([]byte, error) {
	if buf.len == 0 {
		return nil, nil
	}
	if buf.ptr >= uint64(len(data)) || buf.ptr+buf.len > uint64(len(data)) {
		return nil, fmt.Errorf("buffer out of range: ptr=%d len=%d dataLen=%d",
			buf.ptr, buf.len, len(data))
	}
	return data[buf.ptr : buf.ptr+buf.len], nil
}

// readStringBuffer reads a null-terminated string from a resolved buffer.
func readStringBuffer(data []byte, buf resolvedBuffer) (string, error) {
	b, err := readResolvedBuffer(data, buf)
	if err != nil {
		return "", err
	}
	// Strip null terminator if present.
	if len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	return string(b), nil
}

const (
	binderTypePTR          = uint32(0x70742a85)
	binderBufferObjectSize = 40
)

// CreateComponent calls IComponentStore::createComponent() to create
// a component by name.
//
// listenerCookie must be a non-zero cookie returned by RegisterListener
// (which registers a ComponentListenerStub with the hwbinder driver).
// The cookie is used to write a BINDER_TYPE_BINDER flat_binder_object
// that the Codec2 service will use for callbacks.
//
// For the 1.2 store, it calls createComponent_1_2; for 1.0, the base
// createComponent. The pool manager is always null (no BufferPool).
func (s *ComponentStore) CreateComponent(
	ctx context.Context,
	name string,
	listenerCookie uintptr,
) (_ *Component, _err error) {
	logger.Tracef(ctx, "hidlcodec2.CreateComponent(%q)", name)
	defer func() { logger.Tracef(ctx, "/hidlcodec2.CreateComponent: %v", _err) }()

	// Determine which transaction to use based on the store version.
	var txCode binder.TransactionCode
	var fqNameForToken string
	switch s.fqName {
	case storeFQName12:
		txCode = transactionCreateComponent12
		fqNameForToken = storeFQName12
	default:
		txCode = transactionCreateComponent
		fqNameForToken = storeFQName10
	}

	hp := hwparcel.New()
	hp.WriteInterfaceToken(fqNameForToken)

	// Arg 1: hidl_string name.
	hp.WriteHidlString(name)

	// Arg 2: IComponentListener (local binder stub).
	// The kernel will translate this BINDER_TYPE_BINDER into a
	// BINDER_TYPE_HANDLE in the receiving process, allowing the
	// Codec2 service to send callbacks back to us.
	hp.WriteLocalBinder(listenerCookie, listenerCookie)

	// Arg 3: IClientManager pool (null binder).
	hp.WriteNullBinder()

	reply, err := hwservicemanager.TransactHidl(
		ctx, s.transport, s.handle, txCode, hp,
	)
	if err != nil {
		return nil, fmt.Errorf("createComponent transaction: %w", err)
	}
	defer reply.Recycle()

	return parseCreateComponentResponse(s.transport, reply.Data())
}

// parseCreateComponentResponse parses the reply from createComponent.
//
// Wire format:
//
//	[0:4]  int32 HIDL status (0 = OK)
//	[4:8]  int32 C2 Status
//	[8:32] flat_binder_object (24 bytes) containing the IComponent handle
func parseCreateComponentResponse(
	transport binder.Transport,
	data []byte,
) (*Component, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("createComponent response too short: %d bytes", len(data))
	}

	hidlStatus := int32(binary.LittleEndian.Uint32(data[0:]))
	if hidlStatus != 0 {
		return nil, fmt.Errorf("HIDL status error: %d", hidlStatus)
	}

	c2Status := Status(int32(binary.LittleEndian.Uint32(data[4:])))
	if err := c2Status.Err(); err != nil {
		return nil, err
	}

	resp := hwparcel.NewResponseParcel(nil)
	_ = resp

	// Parse the flat_binder_object at offset 8.
	compHandle, err := readBinderHandle(data[8:])
	if err != nil {
		return nil, fmt.Errorf("reading component binder: %w", err)
	}

	return &Component{
		transport: transport,
		handle:    compHandle,
	}, nil
}

// readBinderHandle reads a flat_binder_object (24 bytes) and extracts
// the handle field.
func readBinderHandle(data []byte) (uint32, error) {
	if len(data) < 24 {
		return 0, fmt.Errorf("flat_binder_object too short: %d bytes", len(data))
	}

	objType := binary.LittleEndian.Uint32(data[0:])
	handle := binary.LittleEndian.Uint32(data[8:])

	const binderTypeHandle = uint32(0x73682a85)
	const binderTypeBinder = uint32(0x73622a85)

	switch objType {
	case binderTypeHandle, binderTypeBinder:
		return handle, nil
	default:
		return 0, fmt.Errorf("unexpected binder type %#x", objType)
	}
}
