package pkg

import (
	"github.com/hokaccha/go-prettyjson"
)

// ColorJSON formats and colorizes JSON data
var formatter = prettyjson.NewFormatter()

func init() {
	// Configure the formatter
	formatter.Indent = 2
	formatter.DisabledColor = false
}

// ColorJSON takes any data structure and returns a colorized JSON string
func ColorJSON(v any) string {
	output, err := formatter.Marshal(v)
	if err != nil {
		// Fallback to non-colored string if there's an error
		return "Error formatting JSON"
	}
	return string(output)
}
