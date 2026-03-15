package versionaware

import (
	"encoding/json"
	"os"
	"strconv"
)

// detectAPILevel returns the Android API level of the running device.
// Reads /etc/build_flags.json (world-readable, no root needed, no fork).
// Returns 0 if detection fails (e.g. when running outside Android).
func detectAPILevel() int {
	return detectViaBuildFlags()
}

// buildFlagsPaths lists candidate locations for the build flags file.
var buildFlagsPaths = []string{
	"/etc/build_flags.json",
	"/system/etc/build_flags.json",
}

func detectViaBuildFlags() int {
	for _, path := range buildFlagsPaths {
		n := parseBuildFlags(path)
		if n > 0 {
			return n
		}
	}
	return 0
}

// buildFlagsFile handles two JSON schemas found on Android devices:
//   - API 36+ (Pixel 8a): {"flags": [{"flag_declaration": {...}, "value": {...}}]}
//   - API 35  (emulator): {"flag_artifacts": [{"flag_declaration": {...}, "val": {...}}]}
type buildFlagsFile struct {
	Flags         []buildFlagEntry `json:"flags"`
	FlagArtifacts []buildFlagEntry `json:"flag_artifacts"`
}

type buildFlagEntry struct {
	Declaration buildFlagDeclaration `json:"flag_declaration"`
	// "value" (API 36 schema) or "val" (API 35 schema) — try both.
	Value buildFlagVal `json:"value"`
	Val   buildFlagVal `json:"val"`
}

type buildFlagDeclaration struct {
	Name string `json:"name"`
}

type buildFlagVal struct {
	Val map[string]json.RawMessage `json:"Val"`
}

func parseBuildFlags(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	var file buildFlagsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return 0
	}

	// Merge both arrays — one will be empty depending on the schema.
	entries := append(file.Flags, file.FlagArtifacts...)

	for _, f := range entries {
		if f.Declaration.Name != "RELEASE_PLATFORM_SDK_VERSION" {
			continue
		}

		// Try "value" field first (API 36 schema), then "val" (API 35 schema).
		raw, ok := f.Value.Val["StringValue"]
		if !ok {
			raw, ok = f.Val.Val["StringValue"]
		}
		if !ok {
			return 0
		}

		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return 0
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}
