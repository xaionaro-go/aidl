package spec

// ParamSpec describes a method parameter.
type ParamSpec struct {
	Name        string    `yaml:"name"`
	Type        TypeRef   `yaml:"type"`
	Direction   Direction `yaml:"direction,omitempty"`
	Annotations []string  `yaml:"annotations,omitempty"`
	MinAPILevel int       `yaml:"min_api_level,omitempty"`

	// MaxAPILevel is the last API level where this parameter exists.
	// 0 means "present in all versions from MinAPILevel onward."
	MaxAPILevel int `yaml:"max_api_level,omitempty"`
}
