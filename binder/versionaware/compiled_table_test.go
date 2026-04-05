package versionaware

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AndroidGoLab/binder/binder"
)

func TestCompiledTable_Resolve(t *testing.T) {
	sorted := CompiledTable{
		{Descriptor: "android.app.IBar", Methods: []MethodEntry{
			{Method: "barMethod", Code: binder.FirstCallTransaction + 0},
		}},
		{Descriptor: "android.app.IFoo", Methods: []MethodEntry{
			{Method: "doOther", Code: binder.FirstCallTransaction + 1},
			{Method: "doThing", Code: binder.FirstCallTransaction + 0},
		}},
	}
	require.True(t, sorted.IsSorted())

	assert.Equal(t, binder.FirstCallTransaction+0, sorted.Resolve("android.app.IFoo", "doThing"))
	assert.Equal(t, binder.FirstCallTransaction+1, sorted.Resolve("android.app.IFoo", "doOther"))
	assert.Equal(t, binder.FirstCallTransaction+0, sorted.Resolve("android.app.IBar", "barMethod"))
	assert.Equal(t, binder.TransactionCode(0), sorted.Resolve("android.app.IFoo", "nonExistent"))
	assert.Equal(t, binder.TransactionCode(0), sorted.Resolve("nonexistent.IFoo", "bar"))
}

func TestCompiledTable_ReverseResolve(t *testing.T) {
	table := CompiledTable{
		{Descriptor: "android.app.IFoo", Methods: []MethodEntry{
			{Method: "doOther", Code: binder.FirstCallTransaction + 1},
			{Method: "doThing", Code: binder.FirstCallTransaction + 0},
		}},
	}

	name, ok := table.ReverseResolve("android.app.IFoo", binder.FirstCallTransaction+1)
	assert.True(t, ok)
	assert.Equal(t, "doOther", name)

	_, ok = table.ReverseResolve("android.app.IFoo", binder.FirstCallTransaction+99)
	assert.False(t, ok)

	_, ok = table.ReverseResolve("nonexistent", binder.FirstCallTransaction+0)
	assert.False(t, ok)
}

func TestCompiledTable_IsSorted(t *testing.T) {
	sorted := CompiledTable{
		{Descriptor: "aaa", Methods: []MethodEntry{
			{Method: "aMethod", Code: 1},
			{Method: "bMethod", Code: 2},
		}},
		{Descriptor: "bbb", Methods: []MethodEntry{
			{Method: "xMethod", Code: 1},
		}},
	}
	assert.True(t, sorted.IsSorted())

	unsortedDesc := CompiledTable{
		{Descriptor: "bbb", Methods: nil},
		{Descriptor: "aaa", Methods: nil},
	}
	assert.False(t, unsortedDesc.IsSorted())

	unsortedMethods := CompiledTable{
		{Descriptor: "aaa", Methods: []MethodEntry{
			{Method: "bMethod", Code: 2},
			{Method: "aMethod", Code: 1},
		}},
	}
	assert.False(t, unsortedMethods.IsSorted())

	dupDesc := CompiledTable{
		{Descriptor: "aaa", Methods: nil},
		{Descriptor: "aaa", Methods: nil},
	}
	assert.False(t, dupDesc.IsSorted())
}

func TestCompiledTable_ToVersionTable(t *testing.T) {
	ct := CompiledTable{
		{Descriptor: "android.app.IFoo", Methods: []MethodEntry{
			{Method: "doOther", Code: binder.FirstCallTransaction + 1},
			{Method: "doThing", Code: binder.FirstCallTransaction + 0},
		}},
	}
	vt := ct.ToVersionTable()
	assert.Equal(t, binder.FirstCallTransaction+0, vt["android.app.IFoo"]["doThing"])
	assert.Equal(t, binder.FirstCallTransaction+1, vt["android.app.IFoo"]["doOther"])
}

func TestCompiledTable_HasDescriptor(t *testing.T) {
	ct := CompiledTable{
		{Descriptor: "android.app.IFoo", Methods: nil},
	}
	assert.True(t, ct.HasDescriptor("android.app.IFoo"))
	assert.False(t, ct.HasDescriptor("nonexistent"))
}

func TestCompiledTable_MethodsForDescriptor(t *testing.T) {
	methods := []MethodEntry{
		{Method: "alpha", Code: 1},
		{Method: "beta", Code: 2},
	}
	ct := CompiledTable{
		{Descriptor: "android.app.IFoo", Methods: methods},
	}
	assert.Equal(t, methods, ct.MethodsForDescriptor("android.app.IFoo"))
	assert.Nil(t, ct.MethodsForDescriptor("nonexistent"))
}

func TestTablesAreSorted(t *testing.T) {
	if len(Tables) == 0 {
		t.Skip("Tables not yet populated with CompiledTable data")
	}
	for rev, table := range Tables {
		if !table.IsSorted() {
			t.Errorf("Tables[%q] is not sorted", rev)
		}
	}
}
