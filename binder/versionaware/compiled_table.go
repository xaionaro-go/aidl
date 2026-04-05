package versionaware

import (
	"sort"

	"github.com/AndroidGoLab/binder/binder"
)

// MethodEntry maps a method name to its transaction code.
type MethodEntry struct {
	Method string
	Code   binder.TransactionCode
}

// InterfaceEntry maps a descriptor to its sorted method list.
type InterfaceEntry struct {
	Descriptor string
	Methods    []MethodEntry // sorted by Method
}

// CompiledTable is a read-only table of transaction codes stored as
// sorted slices for zero-cost initialization (no runtime hash table
// construction). Use Resolve for lookups.
type CompiledTable []InterfaceEntry // sorted by Descriptor

// Resolve looks up the transaction code for a method.
// Returns 0 if not found. Uses binary search.
func (t CompiledTable) Resolve(
	descriptor string,
	method string,
) binder.TransactionCode {
	i := sort.Search(len(t), func(i int) bool {
		return t[i].Descriptor >= descriptor
	})
	if i >= len(t) || t[i].Descriptor != descriptor {
		return 0
	}
	methods := t[i].Methods
	j := sort.Search(len(methods), func(j int) bool {
		return methods[j].Method >= method
	})
	if j >= len(methods) || methods[j].Method != method {
		return 0
	}
	return methods[j].Code
}

// ReverseResolve looks up the method name for a transaction code.
// Returns ("", false) if not found.
func (t CompiledTable) ReverseResolve(
	descriptor string,
	code binder.TransactionCode,
) (string, bool) {
	i := sort.Search(len(t), func(i int) bool {
		return t[i].Descriptor >= descriptor
	})
	if i >= len(t) || t[i].Descriptor != descriptor {
		return "", false
	}
	for _, m := range t[i].Methods {
		if m.Code == code {
			return m.Method, true
		}
	}
	return "", false
}

// HasDescriptor reports whether the table contains the given descriptor.
func (t CompiledTable) HasDescriptor(descriptor string) bool {
	i := sort.Search(len(t), func(i int) bool {
		return t[i].Descriptor >= descriptor
	})
	return i < len(t) && t[i].Descriptor == descriptor
}

// MethodsForDescriptor returns the method entries for a descriptor.
// Returns nil if the descriptor is not found.
func (t CompiledTable) MethodsForDescriptor(
	descriptor string,
) []MethodEntry {
	i := sort.Search(len(t), func(i int) bool {
		return t[i].Descriptor >= descriptor
	})
	if i >= len(t) || t[i].Descriptor != descriptor {
		return nil
	}
	return t[i].Methods
}

// ToVersionTable converts a compiled table to a mutable VersionTable.
func (t CompiledTable) ToVersionTable() VersionTable {
	vt := make(VersionTable, len(t))
	for _, iface := range t {
		methods := make(map[string]binder.TransactionCode, len(iface.Methods))
		for _, m := range iface.Methods {
			methods[m.Method] = m.Code
		}
		vt[iface.Descriptor] = methods
	}
	return vt
}

// IsSorted reports whether the table's invariants hold:
// interfaces sorted by Descriptor (strictly increasing),
// methods sorted by Method within each (strictly increasing).
func (t CompiledTable) IsSorted() bool {
	for i := 1; i < len(t); i++ {
		if t[i].Descriptor <= t[i-1].Descriptor {
			return false
		}
	}
	for _, iface := range t {
		for j := 1; j < len(iface.Methods); j++ {
			if iface.Methods[j].Method <= iface.Methods[j-1].Method {
				return false
			}
		}
	}
	return true
}
