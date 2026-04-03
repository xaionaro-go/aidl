package parcel

// This file provides skip functions for Android Notification wire format.
// Used by the native Notification parcelable implementations in both
// android/app and android/app/types packages.

import "fmt"

// SkipNotification reads and discards all fields of a Notification
// from the parcel to advance the position correctly.
func SkipNotification(p *Parcel) error {
	return skipNotification(p)
}

func skipNotification(p *Parcel) error {
	// version
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: version: %w", err)
	}

	// mAllowlistToken (StrongBinder — may be null)
	if _, _, err := p.ReadNullableStrongBinder(); err != nil {
		return fmt.Errorf("notification: allowlistToken: %w", err)
	}

	// when (long)
	if _, err := p.ReadInt64(); err != nil {
		return fmt.Errorf("notification: when: %w", err)
	}

	// creationTime (long)
	if _, err := p.ReadInt64(); err != nil {
		return fmt.Errorf("notification: creationTime: %w", err)
	}

	// mSmallIcon (nullable Icon)
	if err := skipNullableIcon(p); err != nil {
		return fmt.Errorf("notification: smallIcon: %w", err)
	}
	// number (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: number: %w", err)
	}

	// contentIntent (nullable PendingIntent = StrongBinder)
	if err := skipNullablePendingIntent(p); err != nil {
		return fmt.Errorf("notification: contentIntent: %w", err)
	}

	// deleteIntent (nullable PendingIntent)
	if err := skipNullablePendingIntent(p); err != nil {
		return fmt.Errorf("notification: deleteIntent: %w", err)
	}

	// tickerText (nullable CharSequence via TextUtils)
	if err := skipNullableCharSequence(p); err != nil {
		return fmt.Errorf("notification: tickerText: %w", err)
	}

	// tickerView (nullable RemoteViews)
	if err := skipNullableRemoteViews(p); err != nil {
		return fmt.Errorf("notification: tickerView: %w", err)
	}

	// contentView (nullable RemoteViews)
	if err := skipNullableRemoteViews(p); err != nil {
		return fmt.Errorf("notification: contentView: %w", err)
	}

	// mLargeIcon (nullable Icon)
	if err := skipNullableIcon(p); err != nil {
		return fmt.Errorf("notification: largeIcon: %w", err)
	}

	// defaults (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: defaults: %w", err)
	}

	// flags (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: flags: %w", err)
	}

	// sound (nullable Uri)
	if err := skipNullableUri(p); err != nil {
		return fmt.Errorf("notification: sound: %w", err)
	}
	// audioStreamType (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: audioStreamType: %w", err)
	}

	// audioAttributes (nullable AudioAttributes)
	if err := skipNullableAudioAttributes(p); err != nil {
		return fmt.Errorf("notification: audioAttributes: %w", err)
	}
	// vibrate (long array via createLongArray)
	if err := skipLongArray(p); err != nil {
		return fmt.Errorf("notification: vibrate: %w", err)
	}

	// ledARGB, ledOnMS, ledOffMS, iconLevel (4 x int)
	for _, name := range []string{"ledARGB", "ledOnMS", "ledOffMS", "iconLevel"} {
		if _, err := p.ReadInt32(); err != nil {
			return fmt.Errorf("notification: %s: %w", name, err)
		}
	}

	// fullScreenIntent (nullable PendingIntent)
	if err := skipNullablePendingIntent(p); err != nil {
		return fmt.Errorf("notification: fullScreenIntent: %w", err)
	}

	// priority (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: priority: %w", err)
	}

	// category (string8)
	if _, err := p.ReadString(); err != nil {
		return fmt.Errorf("notification: category: %w", err)
	}

	// mGroupKey (string8)
	if _, err := p.ReadString(); err != nil {
		return fmt.Errorf("notification: groupKey: %w", err)
	}

	// mSortKey (string8)
	if _, err := p.ReadString(); err != nil {
		return fmt.Errorf("notification: sortKey: %w", err)
	}

	// extras (Bundle — nullable, has length prefix)
	if err := skipBundle(p); err != nil {
		return fmt.Errorf("notification: extras: %w", err)
	}
	// actions (typed array of Action — nullable)
	if err := skipTypedArrayAction(p); err != nil {
		return fmt.Errorf("notification: actions: %w", err)
	}
	// bigContentView (nullable RemoteViews)
	if err := skipNullableRemoteViews(p); err != nil {
		return fmt.Errorf("notification: bigContentView: %w", err)
	}

	// headsUpContentView (nullable RemoteViews)
	if err := skipNullableRemoteViews(p); err != nil {
		return fmt.Errorf("notification: headsUpContentView: %w", err)
	}

	// visibility (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: visibility: %w", err)
	}

	// publicVersion (nullable Notification — RECURSIVE)
	flag, err := p.ReadInt32()
	if err != nil {
		return fmt.Errorf("notification: publicVersion flag: %w", err)
	}
	if flag != 0 {
		if err := skipNotification(p); err != nil {
			return fmt.Errorf("notification: publicVersion: %w", err)
		}
	}

	// color (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: color: %w", err)
	}

	// mChannelId (nullable string8)
	if err := skipNullableString8(p); err != nil {
		return fmt.Errorf("notification: channelId: %w", err)
	}

	// mTimeout (long)
	if _, err := p.ReadInt64(); err != nil {
		return fmt.Errorf("notification: timeout: %w", err)
	}

	// mShortcutId (nullable string8)
	if err := skipNullableString8(p); err != nil {
		return fmt.Errorf("notification: shortcutId: %w", err)
	}

	// mLocusId (nullable LocusId — writeString(mId))
	if err := skipNullableLocusId(p); err != nil {
		return fmt.Errorf("notification: locusId: %w", err)
	}

	// mBadgeIcon (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: badgeIcon: %w", err)
	}

	// mSettingsText (nullable CharSequence via TextUtils)
	if err := skipNullableCharSequence(p); err != nil {
		return fmt.Errorf("notification: settingsText: %w", err)
	}

	// mGroupAlertBehavior (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: groupAlertBehavior: %w", err)
	}

	// mBubbleMetadata (nullable BubbleMetadata)
	if err := skipNullableBubbleMetadata(p); err != nil {
		return fmt.Errorf("notification: bubbleMetadata: %w", err)
	}

	// mAllowSystemGeneratedContextualActions (boolean — stored as int32)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: allowSystemActions: %w", err)
	}

	// mFgsDeferBehavior (int)
	if _, err := p.ReadInt32(); err != nil {
		return fmt.Errorf("notification: fgsDeferBehavior: %w", err)
	}

	// allPendingIntents (ArraySet<PendingIntent> via writeArraySet)
	if err := skipArraySet(p); err != nil {
		return fmt.Errorf("notification: allPendingIntents: %w", err)
	}
	return nil
}

func skipNullablePendingIntent(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}
	_, _, err = p.ReadNullableStrongBinder()
	return err
}

func skipNullableIcon(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}

	iconType, err := p.ReadInt32()
	if err != nil {
		return fmt.Errorf("icon type: %w", err)
	}

	const (
		iconTypeBitmap         = 1
		iconTypeResource       = 2
		iconTypeData           = 3
		iconTypeURI            = 4
		iconTypeAdaptiveBitmap = 5
		iconTypeURIAdaptive    = 6
	)

	switch iconType {
	case iconTypeBitmap, iconTypeAdaptiveBitmap:
		return fmt.Errorf("cannot skip bitmap icon (type %d)", iconType)
	case iconTypeResource:
		// writeString(pkg), writeInt(resId), writeBoolean(mono), writeFloat(insetScale)
		if _, err := p.ReadString16(); err != nil {
			return err
		}
		p.SetPosition(p.Position() + 12) // resId + mono + insetScale
	case iconTypeData:
		// writeInt(len), writeBlob(data)
		dataLen, err := p.ReadInt32()
		if err != nil {
			return err
		}
		if err := skipBlob(p, dataLen); err != nil {
			return err
		}
	case iconTypeURI, iconTypeURIAdaptive:
		if _, err := p.ReadString16(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown icon type %d", iconType)
	}

	// nullable tintList (ColorStateList)
	tintFlag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if tintFlag != 0 {
		if err := skipColorStateList(p); err != nil {
			return err
		}
	}

	// blendMode (int)
	_, err = p.ReadInt32()
	return err
}

func skipColorStateList(p *Parcel) error {
	count, err := p.ReadInt32()
	if err != nil {
		return err
	}
	for i := int32(0); i < count; i++ {
		arrLen, err := p.ReadInt32()
		if err != nil {
			return err
		}
		if arrLen > 0 {
			p.SetPosition(p.Position() + int(arrLen)*4)
		}
		if _, err := p.ReadInt32(); err != nil {
			return err
		}
	}
	_, err = p.ReadInt32() // defaultColor
	return err
}

func skipNullableCharSequence(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}
	return skipCharSequence(p)
}

func skipCharSequence(p *Parcel) error {
	kind, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if _, err := p.ReadString(); err != nil {
		return err
	}
	if kind == 1 {
		return nil
	}
	// Spanned text: loop reading spans until kind=0.
	for {
		spanType, err := p.ReadInt32()
		if err != nil {
			return err
		}
		if spanType == 0 {
			break
		}
		if err := skipParcelableSpan(p, spanType); err != nil {
			return fmt.Errorf("span type %d: %w", spanType, err)
		}
	}
	return nil
}

func skipParcelableSpan(p *Parcel, spanType int32) error {
	switch spanType {
	case 1: // ALIGNMENT_SPAN
		if _, err := p.ReadString16(); err != nil {
			return err
		}
	case 2, 3, 4, 7, 9, 12: // single int/float spans
		p.SetPosition(p.Position() + 4)
	case 5, 6, 14, 15, 21: // no-data spans
		// nothing
	case 8: // BULLET_SPAN — gapWidth + wantColor + color
		p.SetPosition(p.Position() + 12)
	case 10: // LEADING_MARGIN_SPAN — first + rest
		p.SetPosition(p.Position() + 8)
	case 11, 13: // URL_SPAN / TYPEFACE_SPAN — string
		if _, err := p.ReadString16(); err != nil {
			return err
		}
	case 16: // ABSOLUTE_SIZE_SPAN — size + dip
		p.SetPosition(p.Position() + 8)
	case 17: // TEXT_APPEARANCE_SPAN — complex
		if _, err := p.ReadString16(); err != nil {
			return err
		}
		p.SetPosition(p.Position() + 20)
		if _, err := p.ReadString16(); err != nil {
			return err
		}
		locFlag, err := p.ReadInt32()
		if err != nil {
			return err
		}
		if locFlag != 0 {
			if _, err := p.ReadString16(); err != nil {
				return err
			}
		}
		p.SetPosition(p.Position() + 16)
		if _, err := p.ReadInt32(); err != nil {
			return err
		}
		if _, err := p.ReadInt32(); err != nil {
			return err
		}
		if _, err := p.ReadString16(); err != nil {
			return err
		}
	case 18: // ANNOTATION — key + value
		if _, err := p.ReadString16(); err != nil {
			return err
		}
		if _, err := p.ReadString16(); err != nil {
			return err
		}
	case 22: // LOCALE_SPAN
		if _, err := p.ReadString16(); err != nil {
			return err
		}
	case 24: // LINE_HEIGHT_SPAN
		p.SetPosition(p.Position() + 4)
	case 25: // LINE_BREAK_CONFIG_SPAN
		p.SetPosition(p.Position() + 8)
	default:
		return fmt.Errorf("unknown span type %d", spanType)
	}
	// where() triple: start, end, flags
	p.SetPosition(p.Position() + 12)
	return nil
}

func skipNullableRemoteViews(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}
	return fmt.Errorf("cannot skip non-null RemoteViews")
}

func skipNullableUri(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}
	// Uri.writeToParcel writes int(TYPE_ID) then type-specific data.
	typeID, err := p.ReadInt32()
	if err != nil {
		return err
	}
	return skipUriByType(p, typeID)
}

func skipUriByType(p *Parcel, typeID int32) error {
	switch typeID {
	case 0: // null
		return nil
	case 1, 2, 3: // StringUri, OpaqueUri, HierarchicalUri
		// All Uri types write: writeInt(TYPE_ID) + writeString8(toString())
		// The TYPE_ID was already consumed by the caller.
		_, err := p.ReadString()
		return err
	default:
		return fmt.Errorf("unknown uri type %d", typeID)
	}
}

func skipNullableAudioAttributes(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}
	// mUsage, mContentType, mSource, mFlags
	p.SetPosition(p.Position() + 16)

	parcelFlags, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if parcelFlags&1 == 0 {
		// writeStringArray
		count, err := p.ReadInt32()
		if err != nil {
			return err
		}
		if count >= 0 {
			for i := int32(0); i < count; i++ {
				if _, err := p.ReadString16(); err != nil {
					return err
				}
			}
		}
	} else {
		if _, err := p.ReadString16(); err != nil {
			return err
		}
	}

	marker, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if marker == 1980 {
		return skipBundle(p)
	}
	return nil
}

func skipLongArray(p *Parcel) error {
	count, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if count > 0 {
		p.SetPosition(p.Position() + int(count)*8)
	}
	return nil
}

func skipBundle(p *Parcel) error {
	length, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if length > 0 {
		// Bundle format after length int: int32(magic) + data(length bytes).
		// Skip magic + data.
		p.SetPosition(p.Position() + 4 + int(length))
	}
	return nil
}

func skipTypedArrayAction(p *Parcel) error {
	count, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if count < 0 {
		return nil
	}
	for i := int32(0); i < count; i++ {
		flag, err := p.ReadInt32()
		if err != nil {
			return err
		}
		if flag == 0 {
			continue
		}
		if err := skipAction(p); err != nil {
			return fmt.Errorf("action[%d]: %w", i, err)
		}
	}
	return nil
}

func skipAction(p *Parcel) error {
	if err := skipNullableIcon(p); err != nil {
		return err
	}
	if err := skipCharSequence(p); err != nil {
		return err
	}
	if err := skipNullablePendingIntent(p); err != nil {
		return err
	}
	if err := skipBundle(p); err != nil {
		return err
	}
	if err := skipTypedArrayRemoteInput(p); err != nil {
		return err
	}
	p.SetPosition(p.Position() + 16) // 4 ints
	return nil
}

func skipTypedArrayRemoteInput(p *Parcel) error {
	count, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if count < 0 {
		return nil
	}
	for i := int32(0); i < count; i++ {
		flag, err := p.ReadInt32()
		if err != nil {
			return err
		}
		if flag == 0 {
			continue
		}
		if err := skipRemoteInput(p); err != nil {
			return fmt.Errorf("remoteInput[%d]: %w", i, err)
		}
	}
	return nil
}

func skipRemoteInput(p *Parcel) error {
	// writeString(mResultKey)
	if _, err := p.ReadString16(); err != nil {
		return err
	}
	// writeCharSequence(mLabel)
	if err := skipCharSequence(p); err != nil {
		return err
	}
	// writeCharSequenceArray(mChoices)
	if err := skipCharSequenceArray(p); err != nil {
		return err
	}
	// mFlags + mEditChoicesBeforeSending
	p.SetPosition(p.Position() + 8)
	// writeBundle(mExtras)
	if err := skipBundle(p); err != nil {
		return err
	}
	// writeArraySet(mAllowedDataTypes)
	return skipArraySet(p)
}

func skipCharSequenceArray(p *Parcel) error {
	count, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if count < 0 {
		return nil
	}
	for i := int32(0); i < count; i++ {
		if err := skipCharSequence(p); err != nil {
			return fmt.Errorf("charseq[%d]: %w", i, err)
		}
	}
	return nil
}

func skipNullableString8(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}
	_, err = p.ReadString()
	return err
}

func skipNullableLocusId(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}
	_, err = p.ReadString16()
	return err
}

func skipNullableBubbleMetadata(p *Parcel) error {
	flag, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if flag == 0 {
		return nil
	}
	// 1. nullable PendingIntent
	if err := skipNullablePendingIntent(p); err != nil {
		return err
	}
	// 2. nullable Icon
	if err := skipNullableIcon(p); err != nil {
		return err
	}
	// 3. mDesiredHeight + mFlags
	p.SetPosition(p.Position() + 8)
	// 4. nullable PendingIntent (deleteIntent)
	if err := skipNullablePendingIntent(p); err != nil {
		return err
	}
	// 5. mDesiredHeightResId
	if _, err := p.ReadInt32(); err != nil {
		return err
	}
	// 6. nullable string8 (mShortcutId)
	return skipNullableString8(p)
}

func skipArraySet(p *Parcel) error {
	count, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if count < 0 {
		return nil
	}
	for i := int32(0); i < count; i++ {
		typeTag, err := p.ReadInt32()
		if err != nil {
			return err
		}
		if err := p.SkipWriteValue(typeTag); err != nil {
			return fmt.Errorf("arrayset[%d]: %w", i, err)
		}
	}
	return nil
}

func skipBlob(p *Parcel, _ int32) error {
	blobType, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if blobType == 1 {
		return fmt.Errorf("cannot skip ashmem blob")
	}
	blobLen, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if blobLen > 0 {
		aligned := (int(blobLen) + 3) &^ 3
		p.SetPosition(p.Position() + aligned)
	}
	return nil
}
