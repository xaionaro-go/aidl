package binder

// APILevelFromBinder returns the Android API level from the transport
// backing the given IBinder. Returns 0 if the transport does not
// support API level detection (e.g., raw kernel binder without
// version-aware wrapping).
func APILevelFromBinder(b IBinder) int {
	if b == nil {
		return 0
	}
	t := b.Transport()
	if t == nil {
		return 0
	}
	return t.APILevel()
}
