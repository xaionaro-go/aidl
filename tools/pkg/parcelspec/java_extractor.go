package parcelspec

import (
	"unicode"

	antlr "github.com/antlr4-go/antlr/v4"

	"github.com/xaionaro-go/binder/tools/pkg/javaparser"
)

// ExtractSpecs parses a Java source file and returns ParcelableSpecs
// for each class that contains a writeToParcel method.
func ExtractSpecs(
	javaSrc string,
	packageName string,
) []ParcelableSpec {
	input := antlr.NewInputStream(javaSrc)
	lexer := javaparser.NewJavaLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := javaparser.NewJavaParser(stream)

	// Suppress ANTLR error output during parsing.
	parser.RemoveErrorListeners()

	tree := parser.CompilationUnit()

	listener := newParcelableListener(packageName)
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.specs
}

// javaWriteMethodToSpecType maps Java Parcel write method names
// to their corresponding spec type strings.
var javaWriteMethodToSpecType = map[string]string{
	"writeString8":      "string8",
	"writeString":       "string16",
	"writeString16":     "string16",
	"writeInt":          "int32",
	"writeLong":         "int64",
	"writeFloat":        "float32",
	"writeDouble":       "float64",
	"writeBoolean":      "bool",
	"writeBundle":       "opaque",
	"writeParcelable":   "opaque",
	"writeTypedObject":  "opaque",
	"writeByte":         "int32",
	"writeByteArray":    "opaque",
	"writeBlob":         "opaque",
	"writeStrongBinder": "opaque",
}

// deriveFieldName converts a Java field name to a spec field name
// by stripping the leading "m" prefix convention.
func deriveFieldName(javaFieldName string) string {
	if len(javaFieldName) < 2 {
		return javaFieldName
	}

	// Strip leading "m" prefix if followed by an uppercase letter.
	if javaFieldName[0] == 'm' && unicode.IsUpper(rune(javaFieldName[1])) {
		return javaFieldName[1:]
	}

	// Capitalize first letter.
	return string(unicode.ToUpper(rune(javaFieldName[0]))) + javaFieldName[1:]
}
