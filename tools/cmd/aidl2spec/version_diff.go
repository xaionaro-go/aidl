package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AndroidGoLab/binder/tools/pkg/codegen"
	"github.com/AndroidGoLab/binder/tools/pkg/resolver"
	"github.com/AndroidGoLab/binder/tools/pkg/spec"
)

// diffMethodParams compares oldMethod (from oldAPI) with newMethod (from newAPI)
// and annotates params with MinAPILevel / MaxAPILevel to record cross-version
// differences. The comparison walks params by position:
//
//   - Same type at position N → kept as-is (present in both versions).
//   - Different type at position N → old param gets MaxAPILevel=oldAPI,
//     new param gets MinAPILevel=newAPI. Both appear in the result.
//   - Trailing params only in newMethod → MinAPILevel=newAPI.
//   - Trailing params only in oldMethod → MaxAPILevel=oldAPI.
func diffMethodParams(
	oldMethod spec.MethodSpec,
	newMethod spec.MethodSpec,
	oldAPI int,
	newAPI int,
) []spec.ParamSpec {
	oldParams := oldMethod.Params
	newParams := newMethod.Params

	minLen := len(oldParams)
	if len(newParams) < minLen {
		minLen = len(newParams)
	}

	var result []spec.ParamSpec

	// Walk positions present in both versions.
	for i := 0; i < minLen; i++ {
		if oldParams[i].Type.Equal(newParams[i].Type) {
			result = append(result, newParams[i])
			continue
		}

		// Type changed at this position: emit both variants with version bounds.
		old := oldParams[i]
		old.MaxAPILevel = oldAPI
		result = append(result, old)

		nw := newParams[i]
		nw.MinAPILevel = newAPI
		result = append(result, nw)
	}

	// Trailing params only in the new version.
	for i := minLen; i < len(newParams); i++ {
		p := newParams[i]
		p.MinAPILevel = newAPI
		result = append(result, p)
	}

	// Trailing params only in the old version (removed).
	for i := minLen; i < len(oldParams); i++ {
		p := oldParams[i]
		p.MaxAPILevel = oldAPI
		result = append(result, p)
	}

	return result
}

// paramsChanged reports whether old and new param lists differ in length
// or in the type at any shared position.
func paramsChanged(
	old []spec.ParamSpec,
	new []spec.ParamSpec,
) bool {
	if len(old) != len(new) {
		return true
	}
	for i := range old {
		if !old[i].Type.Equal(new[i].Type) {
			return true
		}
	}
	return false
}

// diffBaselineParams parses AIDL files from the baseline 3rdparty directory,
// converts them to specs, and diffs each interface method's params against
// the current (new) specs. Any trailing params added in the new API get
// MinAPILevel = newAPI.
func diffBaselineParams(
	baseline3rdparty string,
	baselineAPI int,
	newAPI int,
	currentSpecs map[string]*spec.PackageSpec,
) error {
	absBaseline, err := filepath.Abs(baseline3rdparty)
	if err != nil {
		return fmt.Errorf("resolving baseline path: %w", err)
	}

	if _, err := os.Stat(absBaseline); os.IsNotExist(err) {
		return fmt.Errorf("baseline 3rdparty directory not found: %s", absBaseline)
	}

	fmt.Fprintf(os.Stderr, "Diffing params against baseline API %d from %s...\n", baselineAPI, absBaseline)

	aidlFiles, err := discoverAIDLFiles(absBaseline)
	if err != nil {
		return fmt.Errorf("discovering baseline AIDL files: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Baseline: found %d AIDL files\n", len(aidlFiles))

	searchRoots, err := discoverSearchRoots(aidlFiles)
	if err != nil {
		return fmt.Errorf("discovering baseline search roots: %w", err)
	}

	r := resolver.New(searchRoots)
	r.SetSkipUnresolved(true)

	var parseFailCount int
	for _, f := range aidlFiles {
		if err := r.ResolveFile(f); err != nil {
			parseFailCount++
			continue
		}
	}

	allDefs := r.Registry.All()
	baselineSpecs := convertToSpecs(allDefs)
	fmt.Fprintf(os.Stderr, "Baseline: %d packages (%d parse failures)\n", len(baselineSpecs), parseFailCount)

	// Build an index of baseline interface methods by descriptor+method name.
	type methodKey struct {
		goPkg      string
		ifaceName  string
		methodName string
	}
	baselineMethods := map[methodKey]spec.MethodSpec{}
	for goPkg, ps := range baselineSpecs {
		for _, iface := range ps.Interfaces {
			for _, m := range iface.Methods {
				baselineMethods[methodKey{goPkg, iface.Name, m.Name}] = m
			}
		}
	}

	// Diff each method in current specs against baseline.
	var diffCount int
	for goPkg, ps := range currentSpecs {
		for i := range ps.Interfaces {
			iface := &ps.Interfaces[i]
			for j := range iface.Methods {
				m := &iface.Methods[j]
				key := methodKey{
					goPkg:      codegen.AIDLToGoPackage(ps.AIDLPackage),
					ifaceName:  iface.Name,
					methodName: m.Name,
				}
				baselineMethod, ok := baselineMethods[key]
				if !ok {
					// Method not in baseline — all params are new.
					// But we only annotate if the baseline package
					// existed (i.e., the interface was present).
					_, baselinePkgExists := baselineSpecs[goPkg]
					if !baselinePkgExists {
						continue
					}
					// Interface exists in baseline but method doesn't:
					// the method was added in the new API. We don't
					// annotate individual params since the whole method
					// is new (handled by ResolveCode returning an error).
					continue
				}
				if paramsChanged(baselineMethod.Params, m.Params) {
					m.Params = diffMethodParams(baselineMethod, *m, baselineAPI, newAPI)
					diffCount++
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Annotated %d methods with version-dependent params\n", diffCount)
	return nil
}
