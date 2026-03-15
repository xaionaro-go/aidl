package parcelspec

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParcelableSpec_YAMLRoundTrip(t *testing.T) {
	original := ParcelableSpec{
		Package: "android.location",
		Type:    "Location",
		Fields: []FieldSpec{
			{Name: "Provider", Type: "string8"},
			{Name: "FieldsMask", Type: "int32"},
			{Name: "TimeMs", Type: "int64"},
			{Name: "AltitudeMeters", Type: "float64", Condition: "FieldsMask & 1"},
			{Name: "Hidden", Type: "bool"},
			{Name: "Extras", Type: "opaque"},
		},
	}

	data, err := yaml.Marshal(&original)
	require.NoError(t, err)

	var decoded ParcelableSpec
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Equal(t, original, decoded)
}

func TestParcelableSpec_YAMLOmitsEmptyCondition(t *testing.T) {
	spec := ParcelableSpec{
		Package: "com.example",
		Type:    "Simple",
		Fields: []FieldSpec{
			{Name: "Value", Type: "int32"},
		},
	}

	data, err := yaml.Marshal(&spec)
	require.NoError(t, err)

	yamlStr := string(data)
	require.NotContains(t, yamlStr, "condition")
}
