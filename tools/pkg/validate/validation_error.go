package validate

import (
	"fmt"

	"github.com/xaionaro-go/binder/tools/pkg/parser"
)

// ValidationError represents a semantic validation error.
type ValidationError struct {
	Pos     parser.Position
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}
