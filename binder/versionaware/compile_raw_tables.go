package versionaware

import "sort"

// compileRawTables converts a map-based version table collection into
// sorted CompiledTable slices suitable for binary search lookups.
// Used by codes_gen.go to bridge the legacy VersionTable format
// until the code generator emits CompiledTable literals directly.
func compileRawTables(
	raw map[Revision]VersionTable,
) MultiVersionTable {
	result := make(MultiVersionTable, len(raw))
	for rev, vt := range raw {
		result[rev] = compileVersionTable(vt)
	}
	return result
}

// compileVersionTable converts a single VersionTable (map) into
// a sorted CompiledTable (slice).
func compileVersionTable(
	vt VersionTable,
) CompiledTable {
	ct := make(CompiledTable, 0, len(vt))
	for descriptor, methods := range vt {
		entries := make([]MethodEntry, 0, len(methods))
		for method, code := range methods {
			entries = append(entries, MethodEntry{
				Method: method,
				Code:   code,
			})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Method < entries[j].Method
		})
		ct = append(ct, InterfaceEntry{
			Descriptor: descriptor,
			Methods:    entries,
		})
	}
	sort.Slice(ct, func(i, j int) bool {
		return ct[i].Descriptor < ct[j].Descriptor
	})
	return ct
}
