package main

import (
	"testing"

	"github.com/AndroidGoLab/binder/tools/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffMethodParams_TrailingAddition(t *testing.T) {
	oldMethod := spec.MethodSpec{
		Name: "registerClient",
		Params: []spec.ParamSpec{
			{Name: "appId", Type: spec.TypeRef{Name: "ParcelUuid"}},
			{Name: "callback", Type: spec.TypeRef{Name: "IBluetoothGattCallback"}},
			{Name: "eatt_support", Type: spec.TypeRef{Name: "boolean"}},
			{Name: "transport", Type: spec.TypeRef{Name: "int"}},
		},
	}
	newMethod := spec.MethodSpec{
		Name: "registerClient",
		Params: []spec.ParamSpec{
			{Name: "appId", Type: spec.TypeRef{Name: "ParcelUuid"}},
			{Name: "callback", Type: spec.TypeRef{Name: "IBluetoothGattCallback"}},
			{Name: "eatt_support", Type: spec.TypeRef{Name: "boolean"}},
			{Name: "transport", Type: spec.TypeRef{Name: "int"}},
			{Name: "attributionSource", Type: spec.TypeRef{Name: "AttributionSource"}},
		},
	}

	result := diffMethodParams(oldMethod, newMethod, 34, 36)
	require.Len(t, result, 5)
	assert.Equal(t, 0, result[0].MinAPILevel)  // appId: present in both
	assert.Equal(t, 0, result[0].MaxAPILevel)
	assert.Equal(t, 0, result[3].MinAPILevel)  // transport: present in both
	assert.Equal(t, 0, result[3].MaxAPILevel)
	assert.Equal(t, 36, result[4].MinAPILevel) // attributionSource: added in 36
	assert.Equal(t, 0, result[4].MaxAPILevel)
}

func TestDiffMethodParams_NoChange(t *testing.T) {
	method := spec.MethodSpec{
		Name: "disconnect",
		Params: []spec.ParamSpec{
			{Name: "clientIf", Type: spec.TypeRef{Name: "int"}},
			{Name: "address", Type: spec.TypeRef{Name: "String"}},
		},
	}

	result := diffMethodParams(method, method, 34, 36)
	require.Len(t, result, 2)
	assert.Equal(t, 0, result[0].MinAPILevel)
	assert.Equal(t, 0, result[0].MaxAPILevel)
	assert.Equal(t, 0, result[1].MinAPILevel)
	assert.Equal(t, 0, result[1].MaxAPILevel)
}

func TestDiffMethodParams_AllNew(t *testing.T) {
	oldMethod := spec.MethodSpec{
		Name: "newMethod",
	}
	newMethod := spec.MethodSpec{
		Name: "newMethod",
		Params: []spec.ParamSpec{
			{Name: "param1", Type: spec.TypeRef{Name: "int"}},
			{Name: "param2", Type: spec.TypeRef{Name: "String"}},
		},
	}

	result := diffMethodParams(oldMethod, newMethod, 34, 36)
	require.Len(t, result, 2)
	assert.Equal(t, 36, result[0].MinAPILevel)
	assert.Equal(t, 0, result[0].MaxAPILevel)
	assert.Equal(t, 36, result[1].MinAPILevel)
	assert.Equal(t, 0, result[1].MaxAPILevel)
}

func TestDiffMethodParams_ParamRemoval(t *testing.T) {
	oldMethod := spec.MethodSpec{
		Name: "configure",
		Params: []spec.ParamSpec{
			{Name: "id", Type: spec.TypeRef{Name: "int"}},
			{Name: "flags", Type: spec.TypeRef{Name: "int"}},
			{Name: "legacy", Type: spec.TypeRef{Name: "String"}},
		},
	}
	newMethod := spec.MethodSpec{
		Name: "configure",
		Params: []spec.ParamSpec{
			{Name: "id", Type: spec.TypeRef{Name: "int"}},
		},
	}

	result := diffMethodParams(oldMethod, newMethod, 34, 36)
	require.Len(t, result, 3)

	// id: unchanged
	assert.Equal(t, "id", result[0].Name)
	assert.Equal(t, 0, result[0].MinAPILevel)
	assert.Equal(t, 0, result[0].MaxAPILevel)

	// flags: removed (only in old)
	assert.Equal(t, "flags", result[1].Name)
	assert.Equal(t, 0, result[1].MinAPILevel)
	assert.Equal(t, 34, result[1].MaxAPILevel)

	// legacy: removed (only in old)
	assert.Equal(t, "legacy", result[2].Name)
	assert.Equal(t, 0, result[2].MinAPILevel)
	assert.Equal(t, 34, result[2].MaxAPILevel)
}

func TestDiffMethodParams_TypeChange(t *testing.T) {
	oldMethod := spec.MethodSpec{
		Name: "setConfig",
		Params: []spec.ParamSpec{
			{Name: "key", Type: spec.TypeRef{Name: "String"}},
			{Name: "value", Type: spec.TypeRef{Name: "int"}},
		},
	}
	newMethod := spec.MethodSpec{
		Name: "setConfig",
		Params: []spec.ParamSpec{
			{Name: "key", Type: spec.TypeRef{Name: "String"}},
			{Name: "value", Type: spec.TypeRef{Name: "ParcelableConfig"}},
		},
	}

	result := diffMethodParams(oldMethod, newMethod, 34, 36)
	require.Len(t, result, 3)

	// key: unchanged
	assert.Equal(t, "key", result[0].Name)
	assert.Equal(t, "String", result[0].Type.Name)
	assert.Equal(t, 0, result[0].MinAPILevel)
	assert.Equal(t, 0, result[0].MaxAPILevel)

	// value (old variant): capped at oldAPI
	assert.Equal(t, "value", result[1].Name)
	assert.Equal(t, "int", result[1].Type.Name)
	assert.Equal(t, 0, result[1].MinAPILevel)
	assert.Equal(t, 34, result[1].MaxAPILevel)

	// value (new variant): starts at newAPI
	assert.Equal(t, "value", result[2].Name)
	assert.Equal(t, "ParcelableConfig", result[2].Type.Name)
	assert.Equal(t, 36, result[2].MinAPILevel)
	assert.Equal(t, 0, result[2].MaxAPILevel)
}

func TestDiffMethodParams_Mixed(t *testing.T) {
	// Scenario: 4 old params, 3 new params.
	// Position 0: same type (unchanged)
	// Position 1: different type (type change)
	// Position 2: same type (unchanged)
	// Position 3: only in old (removed)
	// No trailing additions in new.
	oldMethod := spec.MethodSpec{
		Name: "complexMethod",
		Params: []spec.ParamSpec{
			{Name: "ctx", Type: spec.TypeRef{Name: "IBinder"}},
			{Name: "config", Type: spec.TypeRef{Name: "int"}},
			{Name: "name", Type: spec.TypeRef{Name: "String"}},
			{Name: "debug", Type: spec.TypeRef{Name: "boolean"}},
		},
	}
	newMethod := spec.MethodSpec{
		Name: "complexMethod",
		Params: []spec.ParamSpec{
			{Name: "ctx", Type: spec.TypeRef{Name: "IBinder"}},
			{Name: "config", Type: spec.TypeRef{Name: "Bundle"}},
			{Name: "name", Type: spec.TypeRef{Name: "String"}},
		},
	}

	result := diffMethodParams(oldMethod, newMethod, 34, 36)

	// Expected: ctx, config(old), config(new), name, debug(removed) = 5 entries.
	require.Len(t, result, 5)

	// ctx: unchanged
	assert.Equal(t, "ctx", result[0].Name)
	assert.Equal(t, "IBinder", result[0].Type.Name)
	assert.Equal(t, 0, result[0].MinAPILevel)
	assert.Equal(t, 0, result[0].MaxAPILevel)

	// config (old variant): capped
	assert.Equal(t, "config", result[1].Name)
	assert.Equal(t, "int", result[1].Type.Name)
	assert.Equal(t, 0, result[1].MinAPILevel)
	assert.Equal(t, 34, result[1].MaxAPILevel)

	// config (new variant): starts at newAPI
	assert.Equal(t, "config", result[2].Name)
	assert.Equal(t, "Bundle", result[2].Type.Name)
	assert.Equal(t, 36, result[2].MinAPILevel)
	assert.Equal(t, 0, result[2].MaxAPILevel)

	// name: unchanged
	assert.Equal(t, "name", result[3].Name)
	assert.Equal(t, "String", result[3].Type.Name)
	assert.Equal(t, 0, result[3].MinAPILevel)
	assert.Equal(t, 0, result[3].MaxAPILevel)

	// debug: removed
	assert.Equal(t, "debug", result[4].Name)
	assert.Equal(t, "boolean", result[4].Type.Name)
	assert.Equal(t, 0, result[4].MinAPILevel)
	assert.Equal(t, 34, result[4].MaxAPILevel)
}

func TestDiffMethodParams_MixedWithTrailingAddition(t *testing.T) {
	// Position 0: same (unchanged)
	// Position 1: different type (type change)
	// Position 2: only in new (trailing addition)
	oldMethod := spec.MethodSpec{
		Name: "update",
		Params: []spec.ParamSpec{
			{Name: "id", Type: spec.TypeRef{Name: "int"}},
			{Name: "data", Type: spec.TypeRef{Name: "byte", IsArray: true}},
		},
	}
	newMethod := spec.MethodSpec{
		Name: "update",
		Params: []spec.ParamSpec{
			{Name: "id", Type: spec.TypeRef{Name: "int"}},
			{Name: "data", Type: spec.TypeRef{Name: "ParcelFileDescriptor"}},
			{Name: "flags", Type: spec.TypeRef{Name: "int"}},
		},
	}

	result := diffMethodParams(oldMethod, newMethod, 34, 36)

	// Expected: id, data(old), data(new), flags = 4 entries.
	require.Len(t, result, 4)

	// id: unchanged
	assert.Equal(t, "id", result[0].Name)
	assert.Equal(t, 0, result[0].MinAPILevel)
	assert.Equal(t, 0, result[0].MaxAPILevel)

	// data (old): capped
	assert.Equal(t, "data", result[1].Name)
	assert.Equal(t, "byte", result[1].Type.Name)
	assert.Equal(t, 0, result[1].MinAPILevel)
	assert.Equal(t, 34, result[1].MaxAPILevel)

	// data (new): starts at newAPI
	assert.Equal(t, "data", result[2].Name)
	assert.Equal(t, "ParcelFileDescriptor", result[2].Type.Name)
	assert.Equal(t, 36, result[2].MinAPILevel)
	assert.Equal(t, 0, result[2].MaxAPILevel)

	// flags: trailing addition
	assert.Equal(t, "flags", result[3].Name)
	assert.Equal(t, 36, result[3].MinAPILevel)
	assert.Equal(t, 0, result[3].MaxAPILevel)
}

func TestDiffMethodParams_GenericTypeArgs(t *testing.T) {
	// Verify that TypeRef.Equal compares TypeArgs recursively.
	oldMethod := spec.MethodSpec{
		Name: "getItems",
		Params: []spec.ParamSpec{
			{
				Name: "items",
				Type: spec.TypeRef{
					Name:     "List",
					TypeArgs: []spec.TypeRef{{Name: "String"}},
				},
			},
		},
	}
	newMethod := spec.MethodSpec{
		Name: "getItems",
		Params: []spec.ParamSpec{
			{
				Name: "items",
				Type: spec.TypeRef{
					Name:     "List",
					TypeArgs: []spec.TypeRef{{Name: "ParcelableItem"}},
				},
			},
		},
	}

	result := diffMethodParams(oldMethod, newMethod, 34, 36)
	require.Len(t, result, 2)

	// old variant: List<String> capped
	assert.Equal(t, "List", result[0].Type.Name)
	assert.Equal(t, "String", result[0].Type.TypeArgs[0].Name)
	assert.Equal(t, 34, result[0].MaxAPILevel)

	// new variant: List<ParcelableItem> starts
	assert.Equal(t, "List", result[1].Type.Name)
	assert.Equal(t, "ParcelableItem", result[1].Type.TypeArgs[0].Name)
	assert.Equal(t, 36, result[1].MinAPILevel)
}

func TestParamsChanged(t *testing.T) {
	t.Run("equal", func(t *testing.T) {
		params := []spec.ParamSpec{
			{Name: "a", Type: spec.TypeRef{Name: "int"}},
			{Name: "b", Type: spec.TypeRef{Name: "String"}},
		}
		assert.False(t, paramsChanged(params, params))
	})

	t.Run("different_length", func(t *testing.T) {
		old := []spec.ParamSpec{
			{Name: "a", Type: spec.TypeRef{Name: "int"}},
		}
		new := []spec.ParamSpec{
			{Name: "a", Type: spec.TypeRef{Name: "int"}},
			{Name: "b", Type: spec.TypeRef{Name: "String"}},
		}
		assert.True(t, paramsChanged(old, new))
	})

	t.Run("different_type", func(t *testing.T) {
		old := []spec.ParamSpec{
			{Name: "a", Type: spec.TypeRef{Name: "int"}},
		}
		new := []spec.ParamSpec{
			{Name: "a", Type: spec.TypeRef{Name: "long"}},
		}
		assert.True(t, paramsChanged(old, new))
	})

	t.Run("both_empty", func(t *testing.T) {
		assert.False(t, paramsChanged(nil, nil))
	})
}
