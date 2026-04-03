package binder

import "os"

// CallerIdentity holds the caller's identity, used to auto-fill
// identity parameters in AIDL proxy method calls.
type CallerIdentity struct {
	PackageName    string
	AttributionTag string
	UserID         int32
	PID            int32
	UID            int32
}

// DefaultCallerIdentity returns the identity for the current process.
// The package name is determined from the UID: UID 2000 maps to
// "com.android.shell", UID 0 (root) maps to "com.android.shell"
// (root can impersonate any package; shell is the most permissive
// non-system identity). Other UIDs default to "com.android.shell"
// as well; callers should override PackageName for app contexts.
func DefaultCallerIdentity() CallerIdentity {
	uid := int32(os.Getuid())
	return CallerIdentity{
		PackageName:    "com.android.shell",
		AttributionTag: "",
		UserID:         0,
		PID:            int32(os.Getpid()),
		UID:            uid,
	}
}
