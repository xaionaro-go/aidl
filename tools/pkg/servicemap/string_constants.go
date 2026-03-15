package servicemap

import (
	antlr "github.com/antlr4-go/antlr/v4"

	"github.com/xaionaro-go/binder/tools/pkg/javaparser"
)

// ExtractStringConstants parses a Java source string and returns a map
// of constant name to string value for all public static final String fields.
func ExtractStringConstants(src string) map[string]string {
	input := antlr.NewInputStream(src)
	lexer := javaparser.NewJavaLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := javaparser.NewJavaParser(stream)

	// Suppress ANTLR error output during parsing.
	parser.RemoveErrorListeners()

	tree := parser.CompilationUnit()

	listener := newStringConstantListener()
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.Constants
}
