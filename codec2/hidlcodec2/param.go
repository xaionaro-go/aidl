package hidlcodec2

import "encoding/binary"

// BuildC2Param constructs a single C2 parameter blob.
//
// C2 param wire format:
//
//	[0:4] uint32 totalSize (= 8 + len(payload))
//	[4:8] uint32 paramIndex
//	[8:]  payload bytes
func BuildC2Param(
	index uint32,
	payload []byte,
) []byte {
	totalSize := 8 + uint32(len(payload))
	buf := make([]byte, totalSize)
	binary.LittleEndian.PutUint32(buf[0:], totalSize)
	binary.LittleEndian.PutUint32(buf[4:], index)
	copy(buf[8:], payload)
	return buf
}

// BuildPictureSizeParam builds a C2StreamPictureSizeInfo parameter.
//
// C2StreamPictureSizeInfo::PARAM_TYPE = 0x4B400000 | (stream << 17).
// Payload: uint32 width, uint32 height.
func BuildPictureSizeParam(
	stream uint32,
	width uint32,
	height uint32,
) []byte {
	index := uint32(0x4B400000) | (stream << 17)
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint32(payload[0:], width)
	binary.LittleEndian.PutUint32(payload[4:], height)
	return BuildC2Param(index, payload)
}

// BuildBitrateParam builds a C2StreamBitrateInfo parameter.
//
// C2StreamBitrateInfo::PARAM_TYPE = 0x4B200000 | (stream << 17).
// Payload: uint32 bitrate.
func BuildBitrateParam(
	stream uint32,
	bitrate uint32,
) []byte {
	index := uint32(0x4B200000) | (stream << 17)
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload[0:], bitrate)
	return BuildC2Param(index, payload)
}

// ConcatParams concatenates multiple C2 param blobs into a single
// byte slice, with 8-byte alignment padding between params as required
// by the Codec2 wire format.
func ConcatParams(params ...[]byte) []byte {
	var total int
	for _, p := range params {
		total += len(p)
		// Add padding to 8-byte alignment.
		if pad := len(p) % 8; pad != 0 {
			total += 8 - pad
		}
	}
	result := make([]byte, 0, total)
	for _, p := range params {
		result = append(result, p...)
		// Pad to 8-byte alignment.
		if pad := len(p) % 8; pad != 0 {
			result = append(result, make([]byte, 8-pad)...)
		}
	}
	return result
}
