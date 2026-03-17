// Package spec defines the intermediate representation for AIDL and Java
// source extraction. Specs are serialized as YAML, one file per Go package.
// Generators read specs to produce Go code, CLI commands, and documentation.
package spec

// PackageSpec is the top-level spec for a single Go package.
// It contains all definitions extracted from AIDL and Java sources
// that belong to this package.
type PackageSpec struct {
	// AIDLPackage is the dot-separated AIDL package name
	// (e.g., "android.app").
	AIDLPackage string `yaml:"package"`

	// GoPackage is the slash-separated Go package path relative to the
	// module root (e.g., "android/app").
	GoPackage string `yaml:"go_package"`

	Interfaces  []InterfaceSpec  `yaml:"interfaces,omitempty"`
	Parcelables []ParcelableSpec `yaml:"parcelables,omitempty"`
	Enums       []EnumSpec       `yaml:"enums,omitempty"`
	Unions      []UnionSpec      `yaml:"unions,omitempty"`

	// Services holds service name → AIDL descriptor mappings
	// extracted from Context.java + SystemServiceRegistry.java.
	// Typically only present in the servicemanager package spec.
	Services []ServiceMapping `yaml:"services,omitempty"`

	// JavaConstants holds typed constant groups extracted from Java sources.
	JavaConstants []JavaConstantGroup `yaml:"java_constants,omitempty"`
}
