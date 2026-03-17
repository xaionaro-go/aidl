package spec

// JavaConstantGroup describes a set of typed constants extracted
// from a Java source file (e.g., LocationProvider constants from
// LocationManager.java).
type JavaConstantGroup struct {
	// Name identifies this constant group (e.g., "LocationProvider").
	Name string `yaml:"name"`

	// GoType is the Go type name for the constants
	// (e.g., "LocationProvider").
	GoType string `yaml:"go_type"`

	Values []JavaConstantValue `yaml:"values"`
}

// JavaConstantValue is a single constant within a group.
type JavaConstantValue struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
