package parcel

import (
	"fmt"
	"math"
)

const (
	// maxListCount is the maximum number of items allowed when reading a typed
	// list from a parcel. This guards against OOM from malicious or corrupted
	// parcels while allowing legitimate large lists.
	maxListCount = 1_000_000
)

// Java Parcel.writeValue type tags.
const (
	valNull             int32 = -1
	valString           int32 = 0
	valInteger          int32 = 1
	valMap              int32 = 2
	valBundle           int32 = 3
	valParcelable       int32 = 4
	valShort            int32 = 5
	valLong             int32 = 6
	valFloat            int32 = 7
	valDouble           int32 = 8
	valBoolean          int32 = 9
	valCharSequence     int32 = 10
	valList             int32 = 11
	valSparseArray      int32 = 12
	valByteArray        int32 = 13
	valStringArray      int32 = 14
	valIBinder          int32 = 15
	valParcelableArray  int32 = 16
	valObjectArray      int32 = 17
	valIntArray         int32 = 18
	valLongArray        int32 = 19
	valByte             int32 = 20
	valSerializable     int32 = 21
	valBooleanArray     int32 = 23
	valCharSequenceArr  int32 = 24
	valPersistableBundle int32 = 25
	valSize             int32 = 26
	valSizeF            int32 = 27
	valDoubleArray      int32 = 28
	valChar             int32 = 29
	valShortArray       int32 = 30
	valCharArray        int32 = 31
	valFloatArray       int32 = 32
)

// isLengthPrefixed returns true if the given writeValue type tag uses a
// length prefix (int32 byte count) after the type tag, matching Java's
// Parcel.isLengthPrefixed().
func isLengthPrefixed(typeTag int32) bool {
	switch typeTag {
	case valMap, valParcelable, valList, valSparseArray,
		valParcelableArray, valObjectArray, valSerializable:
		return true
	default:
		return false
	}
}

// SkipWriteValue reads and discards one value written by Java's
// Parcel.writeValue(). The type tag has already been read by the caller and
// is passed in. Returns an error if the value cannot be skipped.
func (p *Parcel) SkipWriteValue(typeTag int32) error {
	if isLengthPrefixed(typeTag) {
		length, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading length for type %d: %w", typeTag, err)
		}
		if length > 0 {
			p.SetPosition(p.Position() + int(length))
		}
		return nil
	}

	switch typeTag {
	case valNull:
		// Nothing to read.
	case valString:
		_, err := p.ReadString16()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading string: %w", err)
		}
	case valInteger:
		_, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading int: %w", err)
		}
	case valShort, valByte, valBoolean, valChar:
		// All stored as int32 on the wire.
		_, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading small type %d: %w", typeTag, err)
		}
	case valLong:
		_, err := p.ReadInt64()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading long: %w", err)
		}
	case valFloat:
		_, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading float: %w", err)
		}
	case valDouble:
		_, err := p.ReadInt64()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading double: %w", err)
		}
	case valBundle:
		// Bundle: int32 length + data.
		length, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading bundle length: %w", err)
		}
		if length > 0 {
			p.SetPosition(p.Position() + int(length))
		}
	case valByteArray:
		length, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading byte array length: %w", err)
		}
		if length > 0 {
			// Byte arrays are padded to 4-byte alignment.
			aligned := (int(length) + 3) &^ 3
			p.SetPosition(p.Position() + aligned)
		}
	case valIntArray:
		length, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading int array length: %w", err)
		}
		if length > 0 {
			p.SetPosition(p.Position() + int(length)*4)
		}
	case valLongArray, valDoubleArray:
		length, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading long/double array length: %w", err)
		}
		if length > 0 {
			p.SetPosition(p.Position() + int(length)*8)
		}
	case valStringArray:
		count, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading string array count: %w", err)
		}
		for i := int32(0); i < count; i++ {
			if _, err := p.ReadString16(); err != nil {
				return fmt.Errorf("parcel: SkipWriteValue: reading string array[%d]: %w", i, err)
			}
		}
	case valBooleanArray:
		length, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading bool array length: %w", err)
		}
		if length > 0 {
			p.SetPosition(p.Position() + int(length)*4)
		}
	case valFloatArray:
		length, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading float array length: %w", err)
		}
		if length > 0 {
			p.SetPosition(p.Position() + int(length)*4)
		}
	case valShortArray, valCharArray:
		length, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading short/char array length: %w", err)
		}
		if length > 0 {
			// Short and char arrays are written as int32 per element.
			p.SetPosition(p.Position() + int(length)*4)
		}
	case valCharSequence:
		// CharSequence: written via TextUtils.writeToParcel.
		// int32 kind, then kind-specific data. For simplicity,
		// treat as a string16 (covers the common String case).
		_, err := p.ReadString16()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteValue: reading charsequence: %w", err)
		}
	case valSize:
		// Size: width + height (2 x int32).
		p.SetPosition(p.Position() + 8)
	case valSizeF:
		// SizeF: width + height (2 x float = 2 x int32).
		p.SetPosition(p.Position() + 8)
	case valIBinder:
		return fmt.Errorf("parcel: SkipWriteValue: cannot skip IBinder (type 15)")
	default:
		return fmt.Errorf("parcel: SkipWriteValue: unknown type tag %d", typeTag)
	}
	return nil
}

// SkipWriteList reads and discards a list written by Java's
// Parcel.writeList(). Format: int32 count (or -1 for null), then for
// each element a writeValue() entry (int32 type tag + type-specific data).
func (p *Parcel) SkipWriteList() error {
	count, err := p.ReadInt32()
	if err != nil {
		return fmt.Errorf("parcel: SkipWriteList: reading count: %w", err)
	}
	if count < 0 {
		return nil // null list
	}
	if count > int32(maxListCount) {
		return fmt.Errorf("parcel: SkipWriteList: count %d exceeds limit %d", count, maxListCount)
	}
	for i := int32(0); i < count; i++ {
		typeTag, err := p.ReadInt32()
		if err != nil {
			return fmt.Errorf("parcel: SkipWriteList[%d]: reading type tag: %w", i, err)
		}
		if err := p.SkipWriteValue(typeTag); err != nil {
			return fmt.Errorf("parcel: SkipWriteList[%d]: %w", i, err)
		}
	}
	return nil
}

// WriteTypedList writes a list of Parcelable items.
// Writes int32 count (or -1 for nil slice), then marshals each item.
func WriteTypedList[T Parcelable](
	p *Parcel,
	items []T,
) error {
	if items == nil {
		p.WriteInt32(-1)
		return nil
	}

	if len(items) > math.MaxInt32 {
		return fmt.Errorf("parcel: WriteTypedList count %d exceeds int32 max", len(items))
	}

	p.WriteInt32(int32(len(items)))
	for i, item := range items {
		// Non-null indicator (1) before each element, matching AOSP's
		// writeTypedList which writes 1 for present elements.
		p.WriteInt32(1)
		if err := item.MarshalParcel(p); err != nil {
			return fmt.Errorf("parcel: marshaling list item %d: %w", i, err)
		}
	}

	return nil
}

// ReadTypedList reads a list of Parcelable items using the provided factory
// to create new instances. Returns nil if the count is -1.
func ReadTypedList[T Parcelable](
	p *Parcel,
	factory func() T,
) ([]T, error) {
	count, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}

	if count < 0 {
		return nil, nil
	}

	if int(count) > maxListCount {
		return nil, fmt.Errorf("parcel: ReadTypedList count %d exceeds limit %d", count, maxListCount)
	}

	items := make([]T, count)
	for i := range items {
		// Read per-element null indicator, matching AOSP's readTypedList.
		// 0 means null element (skip), non-zero means present.
		nullInd, err := p.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("parcel: reading list item %d null indicator: %w", i, err)
		}
		if nullInd == 0 {
			continue
		}

		items[i] = factory()
		if err := items[i].UnmarshalParcel(p); err != nil {
			return nil, fmt.Errorf("parcel: unmarshaling list item %d: %w", i, err)
		}
	}

	return items, nil
}
