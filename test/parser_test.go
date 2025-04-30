package test

import (
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestParserFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]any
		isError  bool
	}{
		{
			name:  "Parse Simple ID",
			input: "5",
			expected: map[string]any{
				"id": 5,
			},
			isError: false,
		},
		{
			name:  "Parse Simple Object",
			input: "{name: 'John', age: 30}",
			expected: map[string]any{
				"name": "John",
				"age":  30,
			},
			isError: false,
		},
		{
			name:  "Parse Array Values",
			input: "{id: 1}",
			expected: map[string]any{
				"id": 1,
			},
			isError: false,
		},
		{
			name:  "Parse Range",
			input: "{id: (1, 10)}",
			expected: map[string]any{
				"id": map[string]any{
					"range": []int{1, 10},
				},
			},
			isError: false,
		},
		{
			name:  "Parse Multiple Field Assignment",
			input: "{[name, title] = 'Test'}",
			expected: map[string]any{
				"name":  "Test",
				"title": "Test",
			},
			isError: false,
		},
		{
			name:     "Parse Invalid Input",
			input:    "invalid",
			expected: nil,
			isError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := pkg.ParseArg(tc.input)

			if tc.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
