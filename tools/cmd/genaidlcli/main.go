// Command genaidlcli scans generated AIDL proxy Go files and produces
// registry_gen.go and commands_gen.go for the aidlcli tool.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	modulePath = "github.com/xaionaro-go/aidl"
	outputDir  = "tools/cmd/aidlcli"
)

// scanDirs lists the directories (relative to module root) that contain
// generated proxy Go files.
var scanDirs = []string{
	"android",
	"com",
	"src",
	"fuzztest",
	"libgui_test_server",
}

// primitiveTypes maps Go type names to cobra flag helpers.
// Only methods with all parameters in this set get generated commands.
var primitiveTypes = map[string]flagInfo{
	"string":  {FlagMethod: "String", GetMethod: "GetString", ZeroVal: `""`},
	"bool":    {FlagMethod: "Bool", GetMethod: "GetBool", ZeroVal: "false"},
	"int32":   {FlagMethod: "Int32", GetMethod: "GetInt32", ZeroVal: "0"},
	"int64":   {FlagMethod: "Int64", GetMethod: "GetInt64", ZeroVal: "0"},
	"float32": {FlagMethod: "Float32", GetMethod: "GetFloat32", ZeroVal: "0"},
	"float64": {FlagMethod: "Float64", GetMethod: "GetFloat64", ZeroVal: "0"},
	"byte":    {FlagMethod: "Uint8", GetMethod: "GetUint8", ZeroVal: "0"},
	"uint16":  {FlagMethod: "Uint16", GetMethod: "GetUint16", ZeroVal: "0"},
}

type flagInfo struct {
	FlagMethod string
	GetMethod  string
	ZeroVal    string
}

// interfaceInfo holds parsed metadata for one AIDL interface.
type interfaceInfo struct {
	Descriptor       string
	ProxyConstructor string // e.g. "NewActivityManagerProxy"
	ProxyType        string // e.g. "ActivityManagerProxy"
	ImportPath       string // e.g. "github.com/xaionaro-go/aidl/android/app"
	PkgName          string // e.g. "app"
	Methods          []methodInfo
}

type methodInfo struct {
	Name       string
	Params     []paramInfo // excluding ctx
	ReturnType string      // empty if error-only
}

type paramInfo struct {
	Name string
	Type string
}

func main() {
	rootDir, err := findModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	interfaces := scanAllInterfaces(rootDir)
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Descriptor < interfaces[j].Descriptor
	})

	totalMethods := 0
	commandableMethods := 0
	for _, iface := range interfaces {
		for _, m := range iface.Methods {
			totalMethods++
			if allParamsPrimitive(m) {
				commandableMethods++
			}
		}
	}

	fmt.Printf("Scanned interfaces: %d\n", len(interfaces))
	fmt.Printf("Total methods: %d\n", totalMethods)
	fmt.Printf("Commandable methods (all-primitive params): %d\n", commandableMethods)

	outPath := filepath.Join(rootDir, outputDir)

	if err := writeRegistryGen(outPath, interfaces); err != nil {
		fmt.Fprintf(os.Stderr, "error writing registry_gen.go: %v\n", err)
		os.Exit(1)
	}

	if err := writeCommandsGen(outPath, interfaces); err != nil {
		fmt.Fprintf(os.Stderr, "error writing commands_gen.go: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated registry_gen.go and commands_gen.go")
}

// findModuleRoot walks upward from the current directory to find go.mod.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot find go.mod in any parent directory")
		}
		dir = parent
	}
}

// scanAllInterfaces scans directories for generated proxy files.
func scanAllInterfaces(rootDir string) []*interfaceInfo {
	var result []*interfaceInfo

	for _, dir := range scanDirs {
		absDir := filepath.Join(rootDir, dir)
		if info, err := os.Stat(absDir); err != nil || !info.IsDir() {
			continue
		}
		filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			if iface := parseFile(rootDir, path); iface != nil {
				result = append(result, iface)
			}
			return nil
		})
	}

	// Scan root-level *.go files marked as generated.
	entries, err := os.ReadDir(rootDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			path := filepath.Join(rootDir, entry.Name())
			if iface := parseFile(rootDir, path); iface != nil {
				result = append(result, iface)
			}
		}
	}

	return result
}

// parseFile parses a single Go file and extracts interface metadata if
// it contains a Descriptor constant and a proxy constructor.
func parseFile(rootDir, path string) *interfaceInfo {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	if !isGenerated(f) {
		return nil
	}

	descriptor := findDescriptor(f)
	if descriptor == "" {
		return nil
	}

	proxyConstructor, proxyType := findProxyConstructor(f)
	if proxyConstructor == "" {
		return nil
	}

	relDir, err := filepath.Rel(rootDir, filepath.Dir(path))
	if err != nil {
		return nil
	}

	var importPath, pkgName string
	if relDir == "." {
		importPath = modulePath
		pkgName = f.Name.Name
	} else {
		importPath = modulePath + "/" + filepath.ToSlash(relDir)
		pkgName = f.Name.Name
	}

	methods := extractProxyMethods(f, proxyType)

	return &interfaceInfo{
		Descriptor:       descriptor,
		ProxyConstructor: proxyConstructor,
		ProxyType:        proxyType,
		ImportPath:       importPath,
		PkgName:          pkgName,
		Methods:          methods,
	}
}

// isGenerated checks whether the file has a "Code generated" comment.
func isGenerated(f *ast.File) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "Code generated") {
				return true
			}
		}
	}
	return false
}

// findDescriptor finds the first const that starts with "Descriptor" and
// returns its string value.
func findDescriptor(f *ast.File) string {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || len(vs.Values) == 0 {
				continue
			}
			if !strings.HasPrefix(vs.Names[0].Name, "Descriptor") {
				continue
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			return strings.Trim(lit.Value, `"`)
		}
	}
	return ""
}

// findProxyConstructor finds a function like NewFooProxy(remote binder.IBinder)
// and returns the constructor name and proxy type name.
func findProxyConstructor(f *ast.File) (string, string) {
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil {
			continue
		}
		name := fd.Name.Name
		if !strings.HasPrefix(name, "New") || !strings.HasSuffix(name, "Proxy") {
			continue
		}
		params := fd.Type.Params
		if params == nil || len(params.List) != 1 {
			continue
		}
		if !isBinderIBinder(params.List[0].Type) {
			continue
		}
		proxyType := strings.TrimPrefix(name, "New")
		return name, proxyType
	}
	return "", ""
}

// isBinderIBinder checks if an expression is "binder.IBinder".
func isBinderIBinder(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "binder" && sel.Sel.Name == "IBinder"
}

// extractProxyMethods finds all methods on *ProxyType, skipping AsBinder,
// and extracts parameter and return type info.
func extractProxyMethods(f *ast.File, proxyType string) []methodInfo {
	var methods []methodInfo
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}
		if fd.Name.Name == "AsBinder" {
			continue
		}
		if !receiverIs(fd, proxyType) {
			continue
		}

		params := fd.Type.Params.List
		if len(params) == 0 || !isContextContext(params[0].Type) {
			continue
		}

		m := methodInfo{Name: fd.Name.Name}

		for _, p := range params[1:] {
			typeStr := exprToString(p.Type)
			for _, n := range p.Names {
				m.Params = append(m.Params, paramInfo{Name: n.Name, Type: typeStr})
			}
		}

		m.ReturnType = extractReturnType(fd.Type.Results)
		methods = append(methods, m)
	}
	return methods
}

// receiverIs checks whether a function has a receiver of type *typeName.
func receiverIs(fd *ast.FuncDecl, typeName string) bool {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return false
	}
	star, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == typeName
}

// isContextContext checks if an expression is "context.Context".
func isContextContext(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "context" && sel.Sel.Name == "Context"
}

// exprToString renders an AST expression to a simple string.
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprToString(e.Elt)
		}
		return "[" + exprToString(e.Len) + "]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.BasicLit:
		return e.Value
	case *ast.Ellipsis:
		return "..." + exprToString(e.Elt)
	default:
		return "UNKNOWN"
	}
}

// extractReturnType returns the string representation of the non-error
// return type, or "" if the method returns only error.
func extractReturnType(results *ast.FieldList) string {
	if results == nil {
		return ""
	}
	for _, r := range results.List {
		typeStr := exprToString(r.Type)
		if typeStr == "error" {
			continue
		}
		return typeStr
	}
	return ""
}

// allParamsPrimitive checks whether all parameters of a method are
// primitive types that map to cobra flags.
func allParamsPrimitive(m methodInfo) bool {
	for _, p := range m.Params {
		if _, ok := primitiveTypes[p.Type]; !ok {
			return false
		}
	}
	return true
}

// ---- Generation: registry_gen.go ----

func writeRegistryGen(outDir string, interfaces []*interfaceInfo) error {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by genaidlcli. DO NOT EDIT.\n\n")
	buf.WriteString("package main\n\n")
	buf.WriteString("func init() {\n")
	buf.WriteString("\tgeneratedRegistry = &Registry{\n")
	buf.WriteString("\t\tServices: map[string]*ServiceInfo{\n")

	for _, iface := range interfaces {
		fmt.Fprintf(&buf, "\t\t\t%q: {\n", iface.Descriptor)
		fmt.Fprintf(&buf, "\t\t\t\tDescriptor: %q,\n", iface.Descriptor)
		buf.WriteString("\t\t\t\tMethods: []MethodInfo{\n")
		for _, m := range iface.Methods {
			fmt.Fprintf(&buf, "\t\t\t\t\t{Name: %q", m.Name)
			if len(m.Params) > 0 {
				buf.WriteString(", Params: []ParamInfo{")
				for i, p := range m.Params {
					if i > 0 {
						buf.WriteString(", ")
					}
					fmt.Fprintf(&buf, "{Name: %q, Type: %q}", p.Name, p.Type)
				}
				buf.WriteString("}")
			}
			if m.ReturnType != "" {
				fmt.Fprintf(&buf, ", ReturnType: %q", m.ReturnType)
			}
			buf.WriteString("},\n")
		}
		buf.WriteString("\t\t\t\t},\n")
		buf.WriteString("\t\t\t},\n")
	}

	buf.WriteString("\t\t},\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting registry_gen.go: %w\n\nsource:\n%s", err, buf.String())
	}

	return os.WriteFile(filepath.Join(outDir, "registry_gen.go"), formatted, 0644)
}

// ---- Generation: commands_gen.go ----

// commandableIface pairs an interface with its import alias and commandable methods.
type commandableIface struct {
	iface          *interfaceInfo
	importAlias    string
	commandMethods []methodInfo
}

func writeCommandsGen(outDir string, interfaces []*interfaceInfo) error {
	// Collect commandable interfaces and their import aliases.
	importAliases := map[string]string{} // importPath -> alias

	// Reserve names already used by hardcoded imports and Go builtins.
	aliasCounts := map[string]int{
		"context": 1,
		"fmt":     1,
		"os":      1,
		"cobra":   1,
		"binder":  1,
		"main":    1,
	}

	var commandables []commandableIface

	for _, iface := range interfaces {
		// Skip root-module-level interfaces: they live in the "aidl"
		// package which cannot be imported from this "main" package
		// without a circular-looking import and naming conflict.
		if iface.ImportPath == modulePath {
			continue
		}

		var cmdMethods []methodInfo
		for _, m := range iface.Methods {
			if allParamsPrimitive(m) {
				cmdMethods = append(cmdMethods, m)
			}
		}
		if len(cmdMethods) == 0 {
			continue
		}

		alias, ok := importAliases[iface.ImportPath]
		if !ok {
			alias = uniqueAlias(iface.PkgName, aliasCounts)
			importAliases[iface.ImportPath] = alias
		}

		commandables = append(commandables, commandableIface{
			iface:          iface,
			importAlias:    alias,
			commandMethods: cmdMethods,
		})
	}

	var buf bytes.Buffer
	buf.WriteString("// Code generated by genaidlcli. DO NOT EDIT.\n\n")
	buf.WriteString("//go:build linux\n\n")
	buf.WriteString("package main\n\n")

	writeImports(&buf, importAliases)
	writeAddGeneratedCommands(&buf, commandables)
	writeFindServiceByDescriptorFunc(&buf)

	for _, ci := range commandables {
		writeInterfaceGroup(&buf, ci)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting commands_gen.go: %w\n\nsource:\n%s", err, buf.String())
	}

	return os.WriteFile(filepath.Join(outDir, "commands_gen.go"), formatted, 0644)
}

func writeImports(buf *bytes.Buffer, importAliases map[string]string) {
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"os\"\n")
	buf.WriteString("\n")
	buf.WriteString("\t\"github.com/spf13/cobra\"\n")
	buf.WriteString("\t\"github.com/xaionaro-go/aidl/binder\"\n")

	type importEntry struct {
		path  string
		alias string
	}
	var imports []importEntry
	for path, alias := range importAliases {
		// Skip binder since we already import it, and skip the module root package.
		if path == "github.com/xaionaro-go/aidl/binder" || path == modulePath {
			continue
		}
		imports = append(imports, importEntry{path: path, alias: alias})
	}
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].path < imports[j].path
	})

	if len(imports) > 0 {
		buf.WriteString("\n")
	}
	for _, imp := range imports {
		pkgBase := filepath.Base(imp.path)
		// Always use an explicit alias when the alias differs from the
		// directory basename, which happens when we had to disambiguate.
		if imp.alias == pkgBase {
			fmt.Fprintf(buf, "\t%q\n", imp.path)
		} else {
			fmt.Fprintf(buf, "\t%s %q\n", imp.alias, imp.path)
		}
	}
	buf.WriteString(")\n\n")
}

func writeAddGeneratedCommands(buf *bytes.Buffer, commandables []commandableIface) {
	buf.WriteString("func addGeneratedCommands(root *cobra.Command) {\n")
	for _, ci := range commandables {
		fmt.Fprintf(buf, "\troot.AddCommand(%s())\n", cmdGroupFuncName(ci.iface))
	}
	buf.WriteString("}\n\n")
}

func writeFindServiceByDescriptorFunc(buf *bytes.Buffer) {
	buf.WriteString(`func findServiceByDescriptor(
	ctx context.Context,
	conn *Conn,
	descriptor string,
) (binder.IBinder, error) {
	services, err := conn.SM.ListServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	for _, name := range services {
		svc, err := conn.SM.CheckService(ctx, name)
		if err != nil || svc == nil {
			continue
		}
		desc := queryDescriptor(ctx, svc)
		if desc == descriptor {
			return svc, nil
		}
	}

	return nil, fmt.Errorf("no service with descriptor %q found", descriptor)
}

`)
}

func writeInterfaceGroup(buf *bytes.Buffer, ci commandableIface) {
	iface := ci.iface
	groupFunc := cmdGroupFuncName(iface)

	fmt.Fprintf(buf, "func %s() *cobra.Command {\n", groupFunc)
	fmt.Fprintf(buf, "\tcmd := &cobra.Command{\n")
	fmt.Fprintf(buf, "\t\tUse:   %q,\n", iface.Descriptor)
	fmt.Fprintf(buf, "\t\tShort: %q,\n", iface.Descriptor+" methods")
	fmt.Fprintf(buf, "\t}\n\n")

	for _, m := range ci.commandMethods {
		fmt.Fprintf(buf, "\tcmd.AddCommand(%s())\n", cmdMethodFuncName(iface, m))
	}

	buf.WriteString("\n\treturn cmd\n}\n\n")

	for _, m := range ci.commandMethods {
		writeMethodCmd(buf, iface, ci.importAlias, m)
	}
}

func writeMethodCmd(
	buf *bytes.Buffer,
	iface *interfaceInfo,
	alias string,
	m methodInfo,
) {
	funcName := cmdMethodFuncName(iface, m)
	kebabName := camelToKebab(m.Name)
	shortDesc := buildShortDesc(m)
	hasParams := len(m.Params) > 0

	// Determine the qualifier for the proxy constructor call.
	// If the interface is in the root module package, we can't import it
	// (that would be the aidl package itself, and we're in package main).
	// For root-level packages, we need a special alias.
	qualifier := alias

	fmt.Fprintf(buf, "func %s() *cobra.Command {\n", funcName)

	if hasParams {
		buf.WriteString("\tcmd := &cobra.Command{\n")
	} else {
		buf.WriteString("\treturn &cobra.Command{\n")
	}
	fmt.Fprintf(buf, "\t\tUse:   %q,\n", kebabName)
	fmt.Fprintf(buf, "\t\tShort: %q,\n", shortDesc)

	buf.WriteString("\t\tRunE: func(cmd *cobra.Command, args []string) error {\n")
	buf.WriteString("\t\t\tctx := context.Background()\n\n")
	buf.WriteString("\t\t\tconn, err := OpenConn(ctx, cmd)\n")
	buf.WriteString("\t\t\tif err != nil {\n")
	buf.WriteString("\t\t\t\treturn err\n")
	buf.WriteString("\t\t\t}\n")
	buf.WriteString("\t\t\tdefer conn.Close(ctx)\n\n")

	fmt.Fprintf(buf, "\t\t\tsvc, err := findServiceByDescriptor(ctx, conn, %q)\n", iface.Descriptor)
	buf.WriteString("\t\t\tif err != nil {\n")
	buf.WriteString("\t\t\t\treturn err\n")
	buf.WriteString("\t\t\t}\n\n")

	fmt.Fprintf(buf, "\t\t\tproxy := %s.%s(svc)\n\n", qualifier, iface.ProxyConstructor)

	// Read flags.
	for _, p := range m.Params {
		fi := primitiveTypes[p.Type]
		flagVar := "flag" + capitalize(sanitizeVarName(p.Name))
		fmt.Fprintf(buf, "\t\t\t%s, err := cmd.Flags().%s(%q)\n", flagVar, fi.GetMethod, p.Name)
		buf.WriteString("\t\t\tif err != nil {\n")
		fmt.Fprintf(buf, "\t\t\t\treturn fmt.Errorf(\"reading flag %s: %%w\", err)\n", p.Name)
		buf.WriteString("\t\t\t}\n\n")
	}

	// Call proxy method.
	if m.ReturnType != "" {
		fmt.Fprintf(buf, "\t\t\tresult, err := proxy.%s(ctx", m.Name)
	} else {
		fmt.Fprintf(buf, "\t\t\terr = proxy.%s(ctx", m.Name)
	}
	for _, p := range m.Params {
		flagVar := "flag" + capitalize(sanitizeVarName(p.Name))
		fmt.Fprintf(buf, ", %s", flagVar)
	}
	buf.WriteString(")\n")
	buf.WriteString("\t\t\tif err != nil {\n")
	buf.WriteString("\t\t\t\treturn err\n")
	buf.WriteString("\t\t\t}\n\n")

	// Output result.
	buf.WriteString("\t\t\tmode, _ := cmd.Root().PersistentFlags().GetString(\"format\")\n")
	buf.WriteString("\t\t\tf := NewFormatter(mode, os.Stdout)\n")
	if m.ReturnType != "" {
		buf.WriteString("\t\t\tf.Value(\"result\", result)\n")
	} else {
		buf.WriteString("\t\t\tf.Value(\"status\", \"ok\")\n")
	}

	buf.WriteString("\t\t\treturn nil\n")
	buf.WriteString("\t\t},\n")

	if hasParams {
		buf.WriteString("\t}\n\n")
		for _, p := range m.Params {
			fi := primitiveTypes[p.Type]
			fmt.Fprintf(buf, "\tcmd.Flags().%s(%q, %s, %q)\n",
				fi.FlagMethod, p.Name, fi.ZeroVal, p.Name+" ("+p.Type+")")
			fmt.Fprintf(buf, "\t_ = cmd.MarkFlagRequired(%q)\n", p.Name)
		}
		buf.WriteString("\n\treturn cmd\n")
	} else {
		buf.WriteString("\t}\n")
	}

	buf.WriteString("}\n\n")
}

func buildShortDesc(m methodInfo) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Call %s(", m.Name)
	for i, p := range m.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%s %s", p.Name, p.Type)
	}
	sb.WriteString(")")
	if m.ReturnType != "" {
		fmt.Fprintf(&sb, " -> %s", m.ReturnType)
	}
	return sb.String()
}

// sanitizeVarName ensures a Go variable name doesn't conflict with
// reserved words by escaping trailing underscores.
func sanitizeVarName(name string) string {
	// Param names like "map_" or "type_" from generated code are fine
	// as-is for the flag name, but for the variable we strip trailing underscore.
	return strings.TrimRight(name, "_")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// uniqueAlias generates a unique import alias.
func uniqueAlias(pkgName string, counts map[string]int) string {
	counts[pkgName]++
	if counts[pkgName] == 1 {
		return pkgName
	}
	return fmt.Sprintf("%s%d", pkgName, counts[pkgName])
}

// cmdGroupFuncName returns "newCmdAndroidAppIActivityManager".
func cmdGroupFuncName(iface *interfaceInfo) string {
	return "newCmd" + descriptorToIdent(iface.Descriptor)
}

// cmdMethodFuncName returns "newCmdAndroidAppIActivityManager_IsUserAMonkey".
func cmdMethodFuncName(iface *interfaceInfo, m methodInfo) string {
	return "newCmd" + descriptorToIdent(iface.Descriptor) + "_" + m.Name
}

// descriptorToIdent converts "android.app.IActivityManager" to
// "AndroidAppIActivityManager".
func descriptorToIdent(descriptor string) string {
	parts := strings.Split(descriptor, ".")
	var sb strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		sb.WriteString(strings.ToUpper(p[:1]))
		sb.WriteString(p[1:])
	}
	return sb.String()
}

// camelToKebab converts CamelCase to kebab-case.
func camelToKebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('-')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
