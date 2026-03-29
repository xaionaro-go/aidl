package view

import (
	"fmt"

	"github.com/AndroidGoLab/binder/parcel"
)

// KeyCharacterMap is a native-format parcelable that matches the C++ wire
// format produced by android_view_KeyCharacterMap.cpp (JNI) and
// KeyCharacterMap.cpp (libinput).
//
// JNI envelope (nativeWriteToParcel):
//   int32  deviceId
//   int32  hasMap  (bool written as writeInt, 0 or 1)
//   [if hasMap] KeyCharacterMap::writeToParcel data
//
// KeyCharacterMap::writeToParcel:
//   String8  loadFileName  (int32 length + UTF-8 data + null, padded)
//   int32    type
//   int32    layoutOverlayApplied  (bool via writeBool → writeInt32)
//   int32    numKeys
//   for each key:
//     int32  keyCode
//     int32  label
//     int32  number
//     sentinel loop (int32 1 = behavior follows, 0 = end):
//       int32  metaState
//       int32  character
//       int32  fallbackKeyCode
//       int32  replacementKeyCode
//   int32  numKeyRemapping   + pairs of int32
//   int32  numKeysByScanCode + pairs of int32
//   int32  numKeysByUsageCode+ pairs of int32

type KeyCharacterMap struct {
	DeviceId int32
}

var _ parcel.Parcelable = (*KeyCharacterMap)(nil)

func (s *KeyCharacterMap) MarshalParcel(
	p *parcel.Parcel,
) error {
	// Write JNI envelope: deviceId + hasMap=false.
	// We don't preserve the full native key map data, so write a null map.
	p.WriteInt32(s.DeviceId)
	p.WriteBool(false) // hasMap = false
	return nil
}

// maxKeyCharMapKeys is a safety limit matching the C++ MAX_KEYS constant.
const maxKeyCharMapKeys = 8192

func (s *KeyCharacterMap) UnmarshalParcel(
	p *parcel.Parcel,
) error {
	var err error

	// --- JNI envelope ---
	s.DeviceId, err = p.ReadInt32()
	if err != nil {
		return fmt.Errorf("KeyCharacterMap: reading deviceId: %w", err)
	}

	hasMap, err := p.ReadBool()
	if err != nil {
		return fmt.Errorf("KeyCharacterMap: reading hasMap: %w", err)
	}
	if !hasMap {
		return nil
	}

	// --- KeyCharacterMap::readFromParcel ---

	// loadFileName: written as writeString8 (int32 length + UTF-8 data + null, padded).
	if _, err = p.ReadString(); err != nil {
		return fmt.Errorf("KeyCharacterMap: reading loadFileName: %w", err)
	}

	// type (keyboard type enum)
	if _, err = p.ReadInt32(); err != nil {
		return fmt.Errorf("KeyCharacterMap: reading type: %w", err)
	}

	// layoutOverlayApplied (bool via writeBool -> writeInt32)
	if _, err = p.ReadBool(); err != nil {
		return fmt.Errorf("KeyCharacterMap: reading layoutOverlayApplied: %w", err)
	}

	// numKeys
	numKeys, err := p.ReadInt32()
	if err != nil {
		return fmt.Errorf("KeyCharacterMap: reading numKeys: %w", err)
	}
	if numKeys < 0 || numKeys > maxKeyCharMapKeys {
		return fmt.Errorf("KeyCharacterMap: numKeys %d out of range", numKeys)
	}

	for i := int32(0); i < numKeys; i++ {
		// keyCode
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: key[%d] keyCode: %w", i, err)
		}
		// label
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: key[%d] label: %w", i, err)
		}
		// number
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: key[%d] number: %w", i, err)
		}
		// sentinel-terminated behavior list
		for {
			sentinel, serr := p.ReadInt32()
			if serr != nil {
				return fmt.Errorf("KeyCharacterMap: key[%d] behavior sentinel: %w", i, serr)
			}
			if sentinel == 0 {
				break
			}
			// metaState, character, fallbackKeyCode, replacementKeyCode
			for _, field := range []string{"metaState", "character", "fallbackKeyCode", "replacementKeyCode"} {
				if _, err = p.ReadInt32(); err != nil {
					return fmt.Errorf("KeyCharacterMap: key[%d] behavior %s: %w", i, field, err)
				}
			}
		}
	}

	// numKeyRemapping + pairs
	numKeyRemapping, err := p.ReadInt32()
	if err != nil {
		return fmt.Errorf("KeyCharacterMap: reading numKeyRemapping: %w", err)
	}
	for i := int32(0); i < numKeyRemapping; i++ {
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: keyRemapping[%d] key: %w", i, err)
		}
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: keyRemapping[%d] value: %w", i, err)
		}
	}

	// numKeysByScanCode + pairs
	numKeysByScanCode, err := p.ReadInt32()
	if err != nil {
		return fmt.Errorf("KeyCharacterMap: reading numKeysByScanCode: %w", err)
	}
	for i := int32(0); i < numKeysByScanCode; i++ {
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: keysByScanCode[%d] key: %w", i, err)
		}
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: keysByScanCode[%d] value: %w", i, err)
		}
	}

	// numKeysByUsageCode + pairs
	numKeysByUsageCode, err := p.ReadInt32()
	if err != nil {
		return fmt.Errorf("KeyCharacterMap: reading numKeysByUsageCode: %w", err)
	}
	for i := int32(0); i < numKeysByUsageCode; i++ {
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: keysByUsageCode[%d] key: %w", i, err)
		}
		if _, err = p.ReadInt32(); err != nil {
			return fmt.Errorf("KeyCharacterMap: keysByUsageCode[%d] value: %w", i, err)
		}
	}

	return nil
}
