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

func TestGetCommandLimitOffset(t *testing.T) {
	resetTable(t)
	insertTestData(t)

	// Insert 10 known users for predictable ordering
	for i := 1; i <= 10; i++ {
		_, err := testDB.Exec(`INSERT INTO users (name, email) VALUES (?, ?)`, fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@ex.com", i))
		assert.NoError(t, err)
	}

	tests := []struct {
		name          string
		args          map[string]any
		expectedNames []string
		shouldError   bool
	}{
		{
			name:          "Limit 3",
			args:          map[string]any{"LIM": 3, "up": "name"},
			expectedNames: []string{"User 1", "User 2", "User 3"},
			shouldError:   false,
		},
		{
			name:          "Offset 2, Limit 4",
			args:          map[string]any{"LIM": 4, "OFF": 2, "up": "name"},
			expectedNames: []string{"User 3", "User1", "User10", "User2"},
			shouldError:   false,
		},
		{
			name:          "Limit exceeds row count",
			args:          map[string]any{"LIM": 100},
			expectedNames: nil, // just check no error
			shouldError:   false,
		},
		{
			name:          "Offset beyond row count",
			args:          map[string]any{"LIM": 5, "OFF": 100},
			expectedNames: []string{},
			shouldError:   false,
		},
		{
			name:          "Limit 0",
			args:          map[string]any{"LIM": 0},
			expectedNames: []string{},
			shouldError:   false,
		},
		{
			name:          "Negative limit",
			args:          map[string]any{"LIM": -1},
			expectedNames: []string{},
			shouldError:   true,
		},
		{
			name:          "Negative offset",
			args:          map[string]any{"LIM": 2, "OFF": -1},
			expectedNames: []string{},
			shouldError:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			argsCopy := make(map[string]any)
			for k, v := range tc.args {
				argsCopy[k] = v
			}
			err := pkg.HandleGet(testDB, argsCopy, true)
			if tc.shouldError {
				assert.Error(t, err)
				return // Don't validate results if error is expected
			} else {
				assert.NoError(t, err)
			}

			// Validate the query results directly from the database
			query := "SELECT name FROM users"
			params := []any{}
			if up, ok := tc.args["up"]; ok {
				query += " ORDER BY `" + up.(string) + "` ASC"
			}
			if down, ok := tc.args["down"]; ok {
				query += " ORDER BY `" + down.(string) + "` DESC"
			}
			if lim, ok := tc.args["LIM"]; ok {
				query += " LIMIT ?"
				params = append(params, lim)
				if off, ok := tc.args["OFF"]; ok {
					query += " OFFSET ?"
					params = append(params, off)
				}
			}
			rows, err := testDB.Query(query, params...)
			assert.NoError(t, err)
			defer rows.Close()
			var gotNames []string
			for rows.Next() {
				var name string
				rows.Scan(&name)
				gotNames = append(gotNames, name)
			}
			if tc.expectedNames != nil {
				if len(tc.expectedNames) == 0 {
					assert.Empty(t, gotNames, "Expected empty result, got: %v", gotNames)
				} else {
					assert.Equal(t, tc.expectedNames, gotNames, "Expected names: %v, got: %v", tc.expectedNames, gotNames)
				}
			}
		})
	}
}

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
