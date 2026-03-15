package dex

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// DEX header field offsets.
// Per https://source.android.com/docs/core/runtime/dex-format#header-item
const (
	headerSize = 0x70

	offStringIDsSize = 0x38
	offStringIDsOff  = 0x3C
	offTypeIDsSize   = 0x40
	offTypeIDsOff    = 0x44
	offFieldIDsSize  = 0x50
	offFieldIDsOff   = 0x54
	offClassDefsSize = 0x60
	offClassDefsOff  = 0x64
)

// Size of fixed-length structures in the DEX file.
const (
	stringIDItemSize = 4
	typeIDItemSize   = 4
	fieldIDItemSize  = 8
	classDefItemSize = 32
)

// dexFile provides indexed access to a parsed DEX file's sections.
// Strings are read on demand to avoid loading the entire string table.
type dexFile struct {
	data []byte

	stringIDsSize uint32
	stringIDsOff  uint32
	typeIDsSize   uint32
	typeIDsOff    uint32
	fieldIDsSize  uint32
	fieldIDsOff   uint32
	classDefsSize uint32
	classDefsOff  uint32
}

// parseDEXFile validates the header and extracts section offsets.
func parseDEXFile(data []byte) (*dexFile, error) {
	if len(data) < headerSize {
		return nil, fmt.Errorf("DEX data too short: %d bytes (need at least %d)", len(data), headerSize)
	}

	magic := string(data[0:4])
	if magic != "dex\n" {
		return nil, fmt.Errorf("bad DEX magic: %q", data[0:8])
	}

	f := &dexFile{
		data:          data,
		stringIDsSize: binary.LittleEndian.Uint32(data[offStringIDsSize:]),
		stringIDsOff:  binary.LittleEndian.Uint32(data[offStringIDsOff:]),
		typeIDsSize:   binary.LittleEndian.Uint32(data[offTypeIDsSize:]),
		typeIDsOff:    binary.LittleEndian.Uint32(data[offTypeIDsOff:]),
		fieldIDsSize:  binary.LittleEndian.Uint32(data[offFieldIDsSize:]),
		fieldIDsOff:   binary.LittleEndian.Uint32(data[offFieldIDsOff:]),
		classDefsSize: binary.LittleEndian.Uint32(data[offClassDefsSize:]),
		classDefsOff:  binary.LittleEndian.Uint32(data[offClassDefsOff:]),
	}

	return f, nil
}

// readString reads the MUTF-8 string at the given string_ids index.
// For TRANSACTION_* field names (which are ASCII), reading until the
// null terminator is sufficient.
func (f *dexFile) readString(idx uint32) (string, error) {
	if idx >= f.stringIDsSize {
		return "", fmt.Errorf("string index %d out of range (size=%d)", idx, f.stringIDsSize)
	}

	off := f.stringIDsOff + idx*stringIDItemSize
	if off+4 > uint32(len(f.data)) {
		return "", fmt.Errorf("string_id_item at offset 0x%x out of bounds", off)
	}

	dataOff := binary.LittleEndian.Uint32(f.data[off:])
	if dataOff >= uint32(len(f.data)) {
		return "", fmt.Errorf("string_data_off 0x%x out of bounds", dataOff)
	}

	// Skip the ULEB128-encoded utf16_size.
	pos := dataOff
	_, pos, err := readULEB128(f.data, pos)
	if err != nil {
		return "", fmt.Errorf("reading string utf16_size at 0x%x: %w", dataOff, err)
	}

	// Read until null terminator.
	end := pos
	for end < uint32(len(f.data)) && f.data[end] != 0 {
		end++
	}
	if end >= uint32(len(f.data)) {
		return "", fmt.Errorf("string at offset 0x%x not null-terminated", pos)
	}

	return string(f.data[pos:end]), nil
}

// readTypeDescriptor returns the type descriptor string for the given type_ids index.
func (f *dexFile) readTypeDescriptor(idx uint32) (string, error) {
	if idx >= f.typeIDsSize {
		return "", fmt.Errorf("type index %d out of range (size=%d)", idx, f.typeIDsSize)
	}

	off := f.typeIDsOff + idx*typeIDItemSize
	if off+4 > uint32(len(f.data)) {
		return "", fmt.Errorf("type_id_item at offset 0x%x out of bounds", off)
	}

	descriptorIdx := binary.LittleEndian.Uint32(f.data[off:])
	return f.readString(descriptorIdx)
}

// fieldID represents a parsed field_id_item.
type fieldID struct {
	classIdx uint16
	typeIdx  uint16
	nameIdx  uint32
}

// readFieldID parses the field_id_item at the given field_ids index.
func (f *dexFile) readFieldID(idx uint32) (fieldID, error) {
	if idx >= f.fieldIDsSize {
		return fieldID{}, fmt.Errorf("field index %d out of range (size=%d)", idx, f.fieldIDsSize)
	}

	off := f.fieldIDsOff + idx*fieldIDItemSize
	if off+fieldIDItemSize > uint32(len(f.data)) {
		return fieldID{}, fmt.Errorf("field_id_item at offset 0x%x out of bounds", off)
	}

	return fieldID{
		classIdx: binary.LittleEndian.Uint16(f.data[off:]),
		typeIdx:  binary.LittleEndian.Uint16(f.data[off+2:]),
		nameIdx:  binary.LittleEndian.Uint32(f.data[off+4:]),
	}, nil
}

// stubDescriptorToInterface converts a $Stub class descriptor to the
// AIDL interface's dot-separated name.
//
//	"Landroid/app/IActivityManager$Stub;" -> "android.app.IActivityManager"
func stubDescriptorToInterface(desc string) string {
	// Strip leading 'L' and trailing '$Stub;'.
	s := strings.TrimPrefix(desc, "L")
	s = strings.TrimSuffix(s, "$Stub;")
	return strings.ReplaceAll(s, "/", ".")
}
