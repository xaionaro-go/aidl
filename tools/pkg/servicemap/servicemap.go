package servicemap

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// importRe matches Java import statements and captures the fully qualified class name.
var importRe = regexp.MustCompile(`(?m)^import\s+([\w.]+)\s*;`)

// BuildServiceMap reads Context.java and SystemServiceRegistry.java from the
// given frameworks-base path and returns a map from service name to ServiceMapEntry.
// The returned map is keyed by service name (e.g. "power", "account").
func BuildServiceMap(
	frameworksBasePath string,
) (map[string]ServiceMapEntry, error) {
	contextPath := filepath.Join(frameworksBasePath, "core/java/android/content/Context.java")
	contextSrc, err := os.ReadFile(contextPath)
	if err != nil {
		return nil, fmt.Errorf("reading Context.java: %w", err)
	}

	registryPath := filepath.Join(frameworksBasePath, "core/java/android/app/SystemServiceRegistry.java")
	registrySrc, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("reading SystemServiceRegistry.java: %w", err)
	}

	// constantName -> serviceName (e.g. "POWER_SERVICE" -> "power")
	constants := ExtractContextConstants(string(contextSrc))

	// []Registration with ContextConstant and AIDLInterface
	registrations := ExtractRegistrations(string(registrySrc))

	// simpleName -> fullyQualifiedName (e.g. "IPowerManager" -> "android.os.IPowerManager")
	imports := extractImports(string(registrySrc))

	result := make(map[string]ServiceMapEntry, len(registrations))
	for _, reg := range registrations {
		serviceName, ok := constants[reg.ContextConstant]
		if !ok {
			continue
		}

		descriptor := imports[reg.AIDLInterface]
		if descriptor == "" {
			// The interface might be in the same package (android.app);
			// use the simple name as a fallback descriptor.
			descriptor = reg.AIDLInterface
		}

		result[serviceName] = ServiceMapEntry{
			ServiceName:    serviceName,
			ConstantName:   reg.ContextConstant,
			AIDLInterface:  reg.AIDLInterface,
			AIDLDescriptor: descriptor,
		}
	}

	return result, nil
}

// extractImports parses import statements from a Java source string
// and returns a map from simple class name to fully qualified name.
func extractImports(src string) map[string]string {
	matches := importRe.FindAllStringSubmatch(src, -1)
	result := make(map[string]string, len(matches))
	for _, m := range matches {
		fqn := m[1]
		lastDot := strings.LastIndex(fqn, ".")
		if lastDot < 0 {
			continue
		}
		simpleName := fqn[lastDot+1:]
		result[simpleName] = fqn
	}
	return result
}
