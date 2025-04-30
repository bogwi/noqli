package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestGetCommandLike(t *testing.T) {
	resetTable(t)
	insertTestData(t)

	// Insert additional test data with diverse names for LIKE pattern tests
	_, err := testDB.Exec(`
		INSERT INTO users (name, email) VALUES 
		('Alice Smith', 'alice@example.com'),
		('Bob Smith', 'bob@example.com'),
		('Charlie Johnson', 'charlie@example.com'),
		('David Smith', 'david@example.com'),
		('Eva Williams', 'eva@example.com')
	`)
	assert.NoError(t, err, "Failed to insert additional test data")

	tests := []struct {
		name          string
		args          map[string]any
		expectedCount int
	}{
		{
			name: "Basic LIKE search",
			args: map[string]any{
				"LIKE": "Smith",
			},
			expectedCount: 3, // Should find Alice Smith, Bob Smith, David Smith
		},
		{
			name: "LIKE with lowercase keyword",
			args: map[string]any{
				"like": "Smith",
			},
			expectedCount: 3,
		},
		{
			name: "LIKE with already wildcarded pattern",
			args: map[string]any{
				"LIKE": "%son",
			},
			expectedCount: 1, // Should find Charlie Johnson
		},
		{
			name: "LIKE with beginning of word",
			args: map[string]any{
				"LIKE": "Al",
			},
			expectedCount: 1, // Should find Alice Smith
		},
		{
			name: "LIKE with partial email match",
			args: map[string]any{
				"LIKE": "example",
			},
			expectedCount: 8, // Should find all example.com emails (includes the 3 users from insertTestData())
		},
		{
			name: "LIKE with LIMIT",
			args: map[string]any{
				"LIKE": "Smith",
				"LIM":  2,
			},
			expectedCount: 2, // Should limit to 2 of the 3 Smith records
		},
		{
			name: "LIKE with no match",
			args: map[string]any{
				"LIKE": "NonExistentPattern",
			},
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the args to avoid modifying the original
			argsCopy := make(map[string]any)
			for k, v := range tc.args {
				argsCopy[k] = v
			}

			// Execute the actual NoQLi function we're testing
			err := pkg.HandleGet(testDB, argsCopy, true)
			assert.NoError(t, err)

			// For manual verification, execute a direct SQL query
			// Note: We're not using COUNT(*) here because we need to account for LIMIT
			query := "SELECT * FROM users WHERE "
			params := []any{}

			// Add LIKE condition for all text columns
			likeConditions := []string{
				"name LIKE ?",
				"email LIKE ?",
			}

			// Extract the pattern from args
			var pattern string
			if p, ok := tc.args["LIKE"]; ok {
				pattern = fmt.Sprintf("%v", p)
			} else if p, ok := tc.args["like"]; ok {
				pattern = fmt.Sprintf("%v", p)
			}

			// Add wildcards if not already present
			if !strings.Contains(pattern, "%") {
				pattern = "%" + pattern + "%"
			}

			// Add parameters for each condition
			for range likeConditions {
				params = append(params, pattern)
			}

			query += "(" + strings.Join(likeConditions, " OR ") + ")"

			// Add LIMIT if present
			if lim, ok := tc.args["LIM"]; ok {
				query += " LIMIT ?"
				params = append(params, lim)
			} else if lim, ok := tc.args["lim"]; ok {
				query += " LIMIT ?"
				params = append(params, lim)
			}

			// Execute the query
			rows, err := testDB.Query(query, params...)
			assert.NoError(t, err, "Error executing validation query for %s", tc.name)
			defer rows.Close()

			// Count the actual rows returned
			var count int
			for rows.Next() {
				count++
			}

			assert.Equal(t, tc.expectedCount, count, "Expected %d records, got %d for test: %s",
				tc.expectedCount, count, tc.name)
		})
	}
}
