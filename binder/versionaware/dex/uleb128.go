package dex

import "fmt"

// readULEB128 decodes a ULEB128-encoded unsigned integer at the given
// position in data. Returns the decoded value and the position just
// past the last byte consumed.
func readULEB128(
	data []byte,
	pos uint32,
) (uint32, uint32, error) {
	var result uint32
	var shift uint
	for {
		if pos >= uint32(len(data)) {
			return 0, pos, fmt.Errorf("ULEB128 truncated at offset 0x%x", pos)
		}
		b := data[pos]
		pos++
		result |= uint32(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, pos, nil
		}
		shift += 7
		if shift >= 35 {
			return 0, pos, fmt.Errorf("ULEB128 too large at offset 0x%x", pos)
		}
	}
}
