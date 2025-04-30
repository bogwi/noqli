package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestGetCommand(t *testing.T) {
	resetTable(t)
	insertTestData(t)

	// Insert additional test data with different names for ordering tests
	_, err := testDB.Exec(`
		INSERT INTO users (name, email) VALUES 
		('Alice', 'alice@example.com'),
		('Bob', 'bob@example.com'),
		('Charlie', 'charlie@example.com')
	`)
	assert.NoError(t, err, "Failed to insert additional test data")

	tests := []struct {
		name          string
		args          map[string]any
		expectedCount int
		shouldError   bool
	}{
		{
			name:          "Get All Users",
			args:          nil,
			expectedCount: 6, // Updated count to match the additional inserted data
			shouldError:   false,
		},
		{
			name: "Get User by ID",
			args: map[string]any{
				"id": 1,
			},
			expectedCount: 1,
			shouldError:   false,
		},
		{
			name: "Get Multiple Users by ID",
			args: map[string]any{
				"id": []any{1, 2},
			},
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name: "Get Users by ID Range",
			args: map[string]any{
				"id": map[string]any{
					"range": []int{1, 3},
				},
			},
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name: "Get Non-existent User",
			args: map[string]any{
				"id": 999,
			},
			expectedCount: 0,
			shouldError:   false, // NoQLi doesn't error on no records, just returns "No records found"
		},
		// New test cases for filtering by non-ID columns
		{
			name: "Get User by Email",
			args: map[string]any{
				"email": "user1@example.com",
			},
			expectedCount: 1,
			shouldError:   false,
		},
		{
			name: "Get Multiple Users by Email Array",
			args: map[string]any{
				"email": []any{"user1@example.com", "user2@example.com"},
			},
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name: "Get Users by Multiple Criteria",
			args: map[string]any{
				"name":  "User 1",
				"email": "user1@example.com",
			},
			expectedCount: 1,
			shouldError:   false,
		},
		{
			name: "Get Users by Non-existent Email",
			args: map[string]any{
				"email": "nonexistent@example.com",
			},
			expectedCount: 0,
			shouldError:   false,
		},
		// Test cases for ordering
		{
			name: "Get Users Ordered by Name Ascending (UP)",
			args: map[string]any{
				"up": "name",
			},
			expectedCount: 6,
			shouldError:   false,
		},
		{
			name: "Get Users Ordered by Name Ascending (Uppercase UP)",
			args: map[string]any{
				"UP": "name",
			},
			expectedCount: 6,
			shouldError:   false,
		},
		{
			name: "Get Users Ordered by Name Descending (DOWN)",
			args: map[string]any{
				"down": "name",
			},
			expectedCount: 6,
			shouldError:   false,
		},
		{
			name: "Get Users Ordered by Name Descending (Uppercase DOWN)",
			args: map[string]any{
				"DOWN": "name",
			},
			expectedCount: 6,
			shouldError:   false,
		},
		{
			name: "Get Users Filtered and Ordered",
			args: map[string]any{
				"name": []any{"Alice", "Bob", "Charlie"},
				"up":   "name",
			},
			expectedCount: 3,
			shouldError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := pkg.HandleGet(testDB, tc.args, true)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Validate the query results directly from the database
			if tc.args != nil {
				query := "SELECT COUNT(*) FROM users WHERE 1=1"
				var params []any

				// Add conditions for each argument
				for field, value := range tc.args {
					if sliceVal, ok := value.([]any); ok {
						// Handle array values (IN clause)
						placeholders := make([]string, len(sliceVal))
						for i := range placeholders {
							placeholders[i] = "?"
						}
						query += fmt.Sprintf(" AND `%s` IN (%s)", field, strings.Join(placeholders, ","))
						params = append(params, sliceVal...)
					} else if mapVal, ok := value.(map[string]any); ok {
						// Handle range queries
						if rangeSlice, ok := mapVal["range"].([]int); ok && len(rangeSlice) == 2 {
							query += fmt.Sprintf(" AND `%s` >= ? AND `%s` <= ?", field, field)
							params = append(params, rangeSlice[0], rangeSlice[1])
						}
					} else {
						// Handle simple value
						query += fmt.Sprintf(" AND `%s` = ?", field)
						params = append(params, value)
					}
				}

				var count int
				err := testDB.QueryRow(query, params...).Scan(&count)
				assert.NoError(t, err, "Error executing validation query")
				assert.Equal(t, tc.expectedCount, count, "Expected %d records, got %d for test: %s",
					tc.expectedCount, count, tc.name)
			}
		})
	}
}
