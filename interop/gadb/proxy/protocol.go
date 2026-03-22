package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Wire protocol for TCP transport between the host-side proxy and the
// device-side daemon.
//
// Request format:
//   [uint32 total_len][uint32 descriptor_len][descriptor_bytes][uint32 code][uint32 flags][parcel_data]
//
// Response format:
//   [uint32 total_len][uint32 status_code][reply_parcel_data]
//
// All integers are big-endian.

const (
	// maxMessageLen caps the maximum wire message to prevent
	// unbounded allocations from corrupted or malicious frames.
	maxMessageLen = 16 * 1024 * 1024 // 16 MiB
)

// WriteRequest serializes a transaction request onto w.
func WriteRequest(
	w io.Writer,
	descriptor string,
	code uint32,
	flags uint32,
	data []byte,
) error {
	descBytes := []byte(descriptor)
	// total_len covers: descriptor_len(4) + descriptor + code(4) + flags(4) + data
	totalLen := 4 + len(descBytes) + 4 + 4 + len(data)

	buf := make([]byte, 4+totalLen)
	binary.BigEndian.PutUint32(buf[0:4], uint32(totalLen))
	off := 4
	binary.BigEndian.PutUint32(buf[off:off+4], uint32(len(descBytes)))
	off += 4
	copy(buf[off:off+len(descBytes)], descBytes)
	off += len(descBytes)
	binary.BigEndian.PutUint32(buf[off:off+4], code)
	off += 4
	binary.BigEndian.PutUint32(buf[off:off+4], flags)
	off += 4
	copy(buf[off:], data)

	_, err := w.Write(buf)
	if err != nil {
		return fmt.Errorf("writing request: %w", err)
	}
	return nil
}

// ReadRequest deserializes a transaction request from r.
func ReadRequest(
	r io.Reader,
) (descriptor string, code uint32, flags uint32, data []byte, err error) {
	var totalLen uint32
	if err = binary.Read(r, binary.BigEndian, &totalLen); err != nil {
		return "", 0, 0, nil, fmt.Errorf("reading request total_len: %w", err)
	}
	if totalLen > maxMessageLen {
		return "", 0, 0, nil, fmt.Errorf("request too large: %d bytes (max %d)", totalLen, maxMessageLen)
	}

	payload := make([]byte, totalLen)
	if _, err = io.ReadFull(r, payload); err != nil {
		return "", 0, 0, nil, fmt.Errorf("reading request payload: %w", err)
	}

	if len(payload) < 4 {
		return "", 0, 0, nil, fmt.Errorf("request payload too short for descriptor_len")
	}
	descLen := binary.BigEndian.Uint32(payload[0:4])
	off := uint32(4)

	if off+descLen > uint32(len(payload)) {
		return "", 0, 0, nil, fmt.Errorf("descriptor length %d exceeds payload", descLen)
	}
	descriptor = string(payload[off : off+descLen])
	off += descLen

	// code(4) + flags(4) = 8 bytes minimum remaining
	if off+8 > uint32(len(payload)) {
		return "", 0, 0, nil, fmt.Errorf("request payload too short for code+flags")
	}
	code = binary.BigEndian.Uint32(payload[off : off+4])
	off += 4
	flags = binary.BigEndian.Uint32(payload[off : off+4])
	off += 4

	data = payload[off:]
	return descriptor, code, flags, data, nil
}

// WriteResponse serializes a transaction response onto w.
func WriteResponse(
	w io.Writer,
	statusCode uint32,
	data []byte,
) error {
	// total_len covers: status_code(4) + data
	totalLen := 4 + len(data)

	buf := make([]byte, 4+totalLen)
	binary.BigEndian.PutUint32(buf[0:4], uint32(totalLen))
	binary.BigEndian.PutUint32(buf[4:8], statusCode)
	copy(buf[8:], data)

	_, err := w.Write(buf)
	if err != nil {
		return fmt.Errorf("writing response: %w", err)
	}
	return nil
}

// ReadResponse deserializes a transaction response from r.
func ReadResponse(
	r io.Reader,
) (statusCode uint32, data []byte, err error) {
	var totalLen uint32
	if err = binary.Read(r, binary.BigEndian, &totalLen); err != nil {
		return 0, nil, fmt.Errorf("reading response total_len: %w", err)
	}
	if totalLen > maxMessageLen {
		return 0, nil, fmt.Errorf("response too large: %d bytes (max %d)", totalLen, maxMessageLen)
	}
	if totalLen < 4 {
		return 0, nil, fmt.Errorf("response payload too short for status_code")
	}

	payload := make([]byte, totalLen)
	if _, err = io.ReadFull(r, payload); err != nil {
		return 0, nil, fmt.Errorf("reading response payload: %w", err)
	}

	statusCode = binary.BigEndian.Uint32(payload[0:4])
	data = payload[4:]
	return statusCode, data, nil
}
