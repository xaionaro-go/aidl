package servicemap

// ServiceMapEntry represents a fully resolved service mapping from
// service name to its AIDL interface descriptor.
type ServiceMapEntry struct {
	ServiceName    string // e.g. "activity"
	ConstantName   string // e.g. "ACTIVITY_SERVICE"
	AIDLInterface  string // e.g. "IActivityManager" (simple name)
	AIDLDescriptor string // e.g. "android.app.IActivityManager" (fully qualified)
}
