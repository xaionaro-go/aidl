package dex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadULEB128(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint32
		wantPos uint32
	}{
		{
			name:    "zero",
			data:    []byte{0x00},
			want:    0,
			wantPos: 1,
		},
		{
			name:    "single byte 1",
			data:    []byte{0x01},
			want:    1,
			wantPos: 1,
		},
		{
			name:    "single byte 127",
			data:    []byte{0x7F},
			want:    127,
			wantPos: 1,
		},
		{
			name:    "two bytes 128",
			data:    []byte{0x80, 0x01},
			want:    128,
			wantPos: 2,
		},
		{
			name:    "two bytes 300",
			data:    []byte{0xAC, 0x02},
			want:    300,
			wantPos: 2,
		},
		{
			name:    "five bytes max",
			data:    []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F},
			want:    0xFFFFFFFF,
			wantPos: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotPos, err := readULEB128(tt.data, 0)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantPos, gotPos)
		})
	}
}

func TestReadULEB128_Truncated(t *testing.T) {
	// Continuation bit set but no more data.
	_, _, err := readULEB128([]byte{0x80}, 0)
	assert.Error(t, err)
}

func TestReadULEB128_Empty(t *testing.T) {
	_, _, err := readULEB128([]byte{}, 0)
	assert.Error(t, err)
}
