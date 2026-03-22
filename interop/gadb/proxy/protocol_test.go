package proxy

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolRequestRoundtrip(t *testing.T) {
	tests := []struct {
		name       string
		descriptor string
		code       uint32
		flags      uint32
		data       []byte
	}{
		{
			name:       "empty_data",
			descriptor: "android.os.IServiceManager",
			code:       1,
			flags:      0,
			data:       []byte{},
		},
		{
			name:       "with_parcel_data",
			descriptor: "android.app.IActivityManager",
			code:       42,
			flags:      0x10,
			data:       []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02, 0x03, 0x04},
		},
		{
			name:       "empty_descriptor",
			descriptor: "",
			code:       0,
			flags:      0,
			data:       []byte{0xFF},
		},
		{
			name:       "large_data",
			descriptor: "com.example.ILargeService",
			code:       99,
			flags:      0xFFFF,
			data:       bytes.Repeat([]byte{0xAB}, 4096),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := WriteRequest(&buf, tt.descriptor, tt.code, tt.flags, tt.data)
			require.NoError(t, err)

			gotDesc, gotCode, gotFlags, gotData, err := ReadRequest(&buf)
			require.NoError(t, err)

			assert.Equal(t, tt.descriptor, gotDesc)
			assert.Equal(t, tt.code, gotCode)
			assert.Equal(t, tt.flags, gotFlags)
			assert.Equal(t, tt.data, gotData)
		})
	}
}

func TestProtocolResponseRoundtrip(t *testing.T) {
	tests := []struct {
		name       string
		statusCode uint32
		data       []byte
	}{
		{
			name:       "success_empty",
			statusCode: 0,
			data:       []byte{},
		},
		{
			name:       "success_with_data",
			statusCode: 0,
			data:       []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:       "error_status",
			statusCode: 1,
			data:       []byte{},
		},
		{
			name:       "large_reply",
			statusCode: 0,
			data:       bytes.Repeat([]byte{0xCD}, 8192),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := WriteResponse(&buf, tt.statusCode, tt.data)
			require.NoError(t, err)

			gotStatus, gotData, err := ReadResponse(&buf)
			require.NoError(t, err)

			assert.Equal(t, tt.statusCode, gotStatus)
			assert.Equal(t, tt.data, gotData)
		})
	}
}

func TestProtocolReadRequestTooLarge(t *testing.T) {
	var buf bytes.Buffer
	// Write a total_len that exceeds the maximum.
	err := WriteRequest(&buf, "x", 1, 0, bytes.Repeat([]byte{0}, maxMessageLen+1))
	require.NoError(t, err)

	_, _, _, _, err = ReadRequest(&buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestProtocolReadResponseTooLarge(t *testing.T) {
	var buf bytes.Buffer
	err := WriteResponse(&buf, 0, bytes.Repeat([]byte{0}, maxMessageLen+1))
	require.NoError(t, err)

	_, _, err = ReadResponse(&buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestProtocolMultipleMessages(t *testing.T) {
	var buf bytes.Buffer

	// Write two requests back-to-back.
	require.NoError(t, WriteRequest(&buf, "desc1", 1, 0, []byte{0x01}))
	require.NoError(t, WriteRequest(&buf, "desc2", 2, 1, []byte{0x02, 0x03}))

	desc1, code1, flags1, data1, err := ReadRequest(&buf)
	require.NoError(t, err)
	assert.Equal(t, "desc1", desc1)
	assert.Equal(t, uint32(1), code1)
	assert.Equal(t, uint32(0), flags1)
	assert.Equal(t, []byte{0x01}, data1)

	desc2, code2, flags2, data2, err := ReadRequest(&buf)
	require.NoError(t, err)
	assert.Equal(t, "desc2", desc2)
	assert.Equal(t, uint32(2), code2)
	assert.Equal(t, uint32(1), flags2)
	assert.Equal(t, []byte{0x02, 0x03}, data2)
}

// TestProtocolNilDataConsistency verifies that nil and empty data
// produce equivalent wire bytes (both result in empty data on read).
func TestProtocolNilDataConsistency(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	require.NoError(t, WriteRequest(&buf1, "d", 1, 0, nil))
	require.NoError(t, WriteRequest(&buf2, "d", 1, 0, []byte{}))

	assert.Equal(t, buf1.Bytes(), buf2.Bytes())
}
