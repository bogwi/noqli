package test

import (
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestOutputFormats(t *testing.T) {
	resetTable(t)

	// Insert a test record
	err := pkg.HandleCreate(testDB, map[string]any{
		"name":  "Format Test User",
		"email": "format@example.com",
	}, true) // JSON output
	assert.NoError(t, err)

	// Test JSON output (lowercase commands)
	err = pkg.HandleGet(testDB, nil, true)
	assert.NoError(t, err)

	// Test tabular output (uppercase commands)
	err = pkg.HandleGet(testDB, nil, false)
	assert.NoError(t, err)

	// Test update with JSON output
	err = pkg.HandleUpdate(testDB, map[string]any{
		"id":   1,
		"name": "Updated Format User",
	}, true)
	assert.NoError(t, err)

	// Test update with tabular output
	err = pkg.HandleUpdate(testDB, map[string]any{
		"id":    1,
		"email": "updated@example.com",
	}, false)
	assert.NoError(t, err)

	// Test delete with JSON output
	err = pkg.HandleDelete(testDB, map[string]any{
		"id": 1,
	}, true)
	assert.NoError(t, err)
}
