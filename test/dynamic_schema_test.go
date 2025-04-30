package test

import (
	"database/sql"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestDynamicSchema(t *testing.T) {
	resetTable(t)

	// Try to create a user with fields that don't exist yet
	err := pkg.HandleCreate(testDB, map[string]interface{}{
		"name":     "Dynamic User",
		"email":    "dynamic@example.com",
		"age":      25,
		"location": "New York",
		"active":   true,
	}, true)
	assert.NoError(t, err)

	// Check if columns were created
	columns, err := getColumnsForTest(testDB)
	assert.NoError(t, err)

	// Convert to map for easier checking
	columnMap := make(map[string]bool)
	for _, col := range columns {
		columnMap[col] = true
	}

	// Verify required columns exist
	requiredColumns := []string{"id", "name", "email", "age", "location", "active"}
	for _, col := range requiredColumns {
		assert.True(t, columnMap[col], "Column %s should exist", col)
	}
}

// Helper function to get columns for tests
func getColumnsForTest(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW COLUMNS FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var field, fieldType, null, key, defaultVal, extra sql.NullString
		if err := rows.Scan(&field, &fieldType, &null, &key, &defaultVal, &extra); err != nil {
			return nil, err
		}
		columns = append(columns, field.String)
	}

	return columns, nil
}
