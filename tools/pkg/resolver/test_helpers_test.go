package resolver

import (
	"github.com/xaionaro-go/binder/tools/pkg/parser"
)

func parseTestFile(
	filename string,
) (*parser.Document, error) {
	return parser.ParseFile(filename)
}
