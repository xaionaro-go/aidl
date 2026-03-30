package binder

import "context"

// VersionAwareTransport extends Transport with version-aware
// transaction code resolution. Implemented by versionaware.Transport.
type VersionAwareTransport interface {
	Transport

	// ResolveCode maps an AIDL interface descriptor and method name
	// to the correct TransactionCode for the target device.
	// Returns an error if the method does not exist on the device's
	// API level/revision.
	ResolveCode(
		ctx context.Context,
		descriptor string,
		method string,
	) (TransactionCode, error)

	// APILevel returns the detected Android API level of the device
	// (e.g., 35 for Android 15, 36 for Android 16). Returns 0 if
	// the API level is unknown.
	APILevel() int
}
