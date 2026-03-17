package spec

// ServiceMapping connects an Android service name to its AIDL interface.
// Extracted from Context.java constants and SystemServiceRegistry.java
// registrations.
type ServiceMapping struct {
	// ServiceName is the binder service name registered with ServiceManager
	// (e.g., "activity", "power", "clipboard").
	ServiceName string `yaml:"service_name"`

	// ConstantName is the Java constant name from android.content.Context
	// (e.g., "ACTIVITY_SERVICE", "POWER_SERVICE").
	ConstantName string `yaml:"constant_name"`

	// Descriptor is the fully qualified AIDL interface name
	// (e.g., "android.app.IActivityManager").
	Descriptor string `yaml:"descriptor"`
}
