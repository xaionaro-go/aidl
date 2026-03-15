package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"

	"github.com/xaionaro-go/binder/tools/pkg/parcelspec"
	"gopkg.in/yaml.v3"
)

func main() {
	frameworksBase := flag.String("frameworks-base", "tools/pkg/3rdparty/frameworks-base", "Path to the AOSP frameworks-base directory")
	output := flag.String("output", "parcelspecs", "Output directory for YAML spec files")
	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file")
	memProfile := flag.String("memprofile", "", "Write memory profile to file")
	flag.Parse()

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating CPU profile: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if err := run(*frameworksBase, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating memory profile: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		pprof.WriteHeapProfile(f)
	}
}

func run(
	frameworksBase string,
	outputDir string,
) error {
	generated := 0
	skipped := 0

	err := filepath.Walk(frameworksBase, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}

		packageName := extractPackageName(path)
		if packageName == "" {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		// Quick check: skip files that don't contain writeToParcel.
		if !strings.Contains(string(src), "writeToParcel") {
			return nil
		}

		specs := parcelspec.ExtractSpecs(string(src), packageName)
		for _, spec := range specs {
			if len(spec.Fields) == 0 {
				skipped++
				continue
			}

			if err := writeSpec(outputDir, spec); err != nil {
				return fmt.Errorf("writing spec for %s.%s: %w", spec.Package, spec.Type, err)
			}
			generated++
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking %s: %w", frameworksBase, err)
	}

	fmt.Fprintf(os.Stderr, "genparcelspec: generated %d specs, skipped %d empty specs\n", generated, skipped)
	return nil
}

// extractPackageName derives the Java package name from a file path.
// It looks for a "java/" directory component and uses everything after it
// (minus the filename) as the package path.
//
// For example:
//
//	frameworks-base/core/java/android/location/Location.java → android.location
//	frameworks-base/location/java/android/location/LastLocationRequest.java → android.location
func extractPackageName(path string) string {
	// Normalize separators.
	normalized := filepath.ToSlash(path)

	idx := strings.LastIndex(normalized, "/java/")
	if idx < 0 {
		return ""
	}

	// Everything after "/java/" up to the last "/" is the package directory.
	afterJava := normalized[idx+len("/java/"):]
	lastSlash := strings.LastIndex(afterJava, "/")
	if lastSlash < 0 {
		return ""
	}

	pkgDir := afterJava[:lastSlash]
	return strings.ReplaceAll(pkgDir, "/", ".")
}

// writeSpec writes a single ParcelableSpec to a YAML file at the appropriate path.
func writeSpec(
	outputDir string,
	spec parcelspec.ParcelableSpec,
) error {
	pkgPath := strings.ReplaceAll(spec.Package, ".", "/")
	dir := filepath.Join(outputDir, pkgPath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := yaml.Marshal(&spec)
	if err != nil {
		return fmt.Errorf("marshaling YAML: %w", err)
	}

	outPath := filepath.Join(dir, spec.Type+".yaml")
	return os.WriteFile(outPath, data, 0o644)
}
