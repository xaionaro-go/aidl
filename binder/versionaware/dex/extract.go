package dex

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const transactionFieldPrefix = "TRANSACTION_"

// ExtractFromJAR opens a JAR (ZIP) file, finds all DEX files inside,
// and extracts TRANSACTION_* constants from all $Stub inner classes.
// Returns a map of fully qualified interface name to TransactionCodes.
//
// Example key: "android.app.IActivityManager"
func ExtractFromJAR(path string) (map[string]TransactionCodes, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("opening JAR %q: %w", path, err)
	}
	defer zr.Close()

	result := map[string]TransactionCodes{}

	for _, zf := range zr.File {
		if !strings.HasSuffix(zf.Name, ".dex") {
			continue
		}

		rc, err := zf.Open()
		if err != nil {
			return nil, fmt.Errorf("opening %q in JAR: %w", zf.Name, err)
		}

		data := make([]byte, zf.UncompressedSize64)
		n, err := readFull(rc, data)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("reading %q from JAR: %w", zf.Name, err)
		}
		data = data[:n]

		dexResult, err := ExtractFromDEX(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", zf.Name, err)
		}

		for iface, codes := range dexResult {
			existing, ok := result[iface]
			if !ok {
				result[iface] = codes
				continue
			}
			// Merge (later DEX files can contribute additional classes).
			for method, code := range codes {
				existing[method] = code
			}
		}
	}

	return result, nil
}

// readFull reads from r until EOF or the buffer is full.
// Returns the number of bytes read.
func readFull(r io.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			if errors.Is(err, io.EOF) {
				return total, nil
			}
			return total, err
		}
	}
	return total, nil
}

// ExtractFromDEX parses a single DEX file and extracts TRANSACTION_*
// constants from all $Stub classes.
// Returns a map of fully qualified interface name to TransactionCodes.
func ExtractFromDEX(data []byte) (map[string]TransactionCodes, error) {
	f, err := parseDEXFile(data)
	if err != nil {
		return nil, err
	}

	result := map[string]TransactionCodes{}

	for ci := uint32(0); ci < f.classDefsSize; ci++ {
		off := f.classDefsOff + ci*classDefItemSize
		if off+classDefItemSize > uint32(len(data)) {
			return nil, fmt.Errorf("class_def[%d] at offset 0x%x out of bounds", ci, off)
		}

		classIdx := binary.LittleEndian.Uint32(data[off:])
		desc, err := f.readTypeDescriptor(classIdx)
		if err != nil {
			return nil, fmt.Errorf("class_def[%d]: reading descriptor: %w", ci, err)
		}

		if !strings.HasSuffix(desc, "$Stub;") {
			continue
		}

		codes, err := extractStubTransactions(f, data, off)
		if err != nil {
			return nil, fmt.Errorf("class %s: %w", desc, err)
		}
		if len(codes) == 0 {
			continue
		}

		ifaceName := stubDescriptorToInterface(desc)
		result[ifaceName] = codes
	}

	return result, nil
}

// extractStubTransactions reads the static fields and their initial
// values from a single $Stub class definition, returning only
// TRANSACTION_* integer constants.
func extractStubTransactions(
	f *dexFile,
	data []byte,
	classDefOff uint32,
) (TransactionCodes, error) {
	classDataOff := binary.LittleEndian.Uint32(data[classDefOff+0x18:])
	staticValuesOff := binary.LittleEndian.Uint32(data[classDefOff+0x1C:])

	// A class with no class_data or no static initializer values
	// cannot have TRANSACTION_* constants.
	if classDataOff == 0 || staticValuesOff == 0 {
		return nil, nil
	}

	staticFieldIndices, err := readStaticFieldIndices(data, classDataOff)
	if err != nil {
		return nil, fmt.Errorf("reading class_data: %w", err)
	}
	if len(staticFieldIndices) == 0 {
		return nil, nil
	}

	staticValues, err := readStaticValues(data, staticValuesOff)
	if err != nil {
		return nil, fmt.Errorf("reading static values: %w", err)
	}

	codes := TransactionCodes{}
	for i, fieldIdx := range staticFieldIndices {
		fid, err := f.readFieldID(fieldIdx)
		if err != nil {
			return nil, fmt.Errorf("reading field_id[%d]: %w", fieldIdx, err)
		}

		name, err := f.readString(fid.nameIdx)
		if err != nil {
			return nil, fmt.Errorf("reading field name for field_id[%d]: %w", fieldIdx, err)
		}

		if !strings.HasPrefix(name, transactionFieldPrefix) {
			continue
		}

		if i >= len(staticValues) {
			// The encoded_array_item may have fewer entries than
			// static fields; remaining fields are zero-initialized.
			continue
		}

		methodName := name[len(transactionFieldPrefix):]
		codes[methodName] = uint32(staticValues[i].intVal)
	}

	return codes, nil
}

// readStaticFieldIndices parses the class_data_item at the given offset
// and returns the absolute field_ids indices of all static fields.
func readStaticFieldIndices(
	data []byte,
	classDataOff uint32,
) ([]uint32, error) {
	pos := classDataOff

	staticFieldsSize, pos, err := readULEB128(data, pos)
	if err != nil {
		return nil, fmt.Errorf("reading static_fields_size: %w", err)
	}

	// Skip past the remaining size fields (instance_fields, direct_methods, virtual_methods).
	for i := 0; i < 3; i++ {
		_, pos, err = readULEB128(data, pos)
		if err != nil {
			return nil, fmt.Errorf("reading size field %d: %w", i+1, err)
		}
	}

	// Read static field entries (field_idx_diff + access_flags pairs).
	indices := make([]uint32, 0, staticFieldsSize)
	var fieldIdx uint32
	for i := uint32(0); i < staticFieldsSize; i++ {
		diff, newPos, err := readULEB128(data, pos)
		if err != nil {
			return nil, fmt.Errorf("reading static_field[%d] idx_diff: %w", i, err)
		}
		pos = newPos

		_, pos, err = readULEB128(data, pos) // access_flags
		if err != nil {
			return nil, fmt.Errorf("reading static_field[%d] access_flags: %w", i, err)
		}

		fieldIdx += diff
		indices = append(indices, fieldIdx)
	}

	return indices, nil
}

// readStaticValues parses the encoded_array_item at the given offset
// and returns the decoded values.
func readStaticValues(
	data []byte,
	staticValuesOff uint32,
) ([]encodedValue, error) {
	size, pos, err := readULEB128(data, staticValuesOff)
	if err != nil {
		return nil, fmt.Errorf("reading encoded_array size: %w", err)
	}

	values := make([]encodedValue, 0, size)
	for i := uint32(0); i < size; i++ {
		val, newPos, err := readEncodedValue(data, pos)
		if err != nil {
			return nil, fmt.Errorf("reading encoded_value[%d]: %w", i, err)
		}
		pos = newPos
		values = append(values, val)
	}

	return values, nil
}
