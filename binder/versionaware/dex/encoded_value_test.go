package dex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadEncodedValue_Int(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want int64
	}{
		{
			name: "int 1 byte positive",
			// value_type=0x04 (INT), value_arg=0 → 1 byte of data.
			data: []byte{0x04, 0x2A},
			want: 42,
		},
		{
			name: "int 2 bytes",
			// value_type=0x04, value_arg=1 → 2 bytes. 0x0110 = 272.
			data: []byte{0x24, 0x10, 0x01},
			want: 272,
		},
		{
			name: "int 1 byte negative",
			// value_type=0x04, value_arg=0 → 1 byte. 0xFF = -1 (sign-extended).
			data: []byte{0x04, 0xFF},
			want: -1,
		},
		{
			name: "int 4 bytes",
			// value_type=0x04, value_arg=3 → 4 bytes. 0x0000006E = 110.
			data: []byte{0x64, 0x6E, 0x00, 0x00, 0x00},
			want: 110,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, pos, err := readEncodedValue(tt.data, 0)
			require.NoError(t, err)
			assert.Equal(t, tt.want, val.intVal)
			assert.Equal(t, uint32(len(tt.data)), pos)
		})
	}
}

func TestReadEncodedValue_Null(t *testing.T) {
	data := []byte{0x1e} // NULL, value_arg=0
	val, pos, err := readEncodedValue(data, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val.intVal)
	assert.Equal(t, uint32(1), pos)
}

func TestReadEncodedValue_Boolean(t *testing.T) {
	// BOOLEAN true: value_type=0x1f, value_arg=1 → byte = (1<<5)|0x1f = 0x3f
	data := []byte{0x3f}
	val, pos, err := readEncodedValue(data, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val.intVal)
	assert.Equal(t, uint32(1), pos)

	// BOOLEAN false: value_type=0x1f, value_arg=0 → byte = 0x1f
	data = []byte{0x1f}
	val, pos, err = readEncodedValue(data, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val.intVal)
	assert.Equal(t, uint32(1), pos)
}

func TestReadEncodedValue_Byte(t *testing.T) {
	// BYTE: value_type=0x00, value_arg=0, 1 byte data.
	// value 0x80 = -128 (signed).
	data := []byte{0x00, 0x80}
	val, pos, err := readEncodedValue(data, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(-128), val.intVal)
	assert.Equal(t, uint32(2), pos)
}

func TestReadEncodedValue_Short(t *testing.T) {
	// SHORT: value_type=0x02, value_arg=1 → 2 bytes.
	// 0xFF 0x7F = 0x7FFF = 32767.
	data := []byte{0x22, 0xFF, 0x7F}
	val, pos, err := readEncodedValue(data, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(32767), val.intVal)
	assert.Equal(t, uint32(3), pos)
}

func TestReadEncodedValue_String(t *testing.T) {
	// STRING: value_type=0x17, value_arg=0 → 1 byte index.
	data := []byte{0x17, 0x05}
	val, pos, err := readEncodedValue(data, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(5), val.intVal) // string_ids index
	assert.Equal(t, uint32(2), pos)
}

func TestReadEncodedValue_Truncated(t *testing.T) {
	_, _, err := readEncodedValue([]byte{}, 0)
	assert.Error(t, err, "empty data should fail")

	// INT with value_arg=3 (4 bytes) but only 2 bytes of data.
	_, _, err = readEncodedValue([]byte{0x64, 0x01, 0x02}, 0)
	assert.Error(t, err, "truncated int should fail")
}
