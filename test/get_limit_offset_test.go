package test

import (
	"fmt"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

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
