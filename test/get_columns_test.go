package test

import (
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestGetCommandColumns(t *testing.T) {
	resetTable(t)

	// Insert test data
	_, err := testDB.Exec(`
		INSERT INTO users (name, email, status, category, priority) VALUES 
		('User 1', 'user1@example.com', 'active', 'customer', 'high'),
		('User 2', 'user2@example.com', 'active', 'customer', 'medium'),
		('User 3', 'user3@example.com', 'active', 'vendor', 'low'),
		('Alice', 'alice@example.com', 'pending', 'customer', 'medium'),
		('Bob', 'bob@example.com', 'inactive', 'vendor', 'low')
	`)
	assert.NoError(t, err, "Failed to insert test data")

	tests := []struct {
		name          string
		commandStr    string
		expectedCount int
		expectedCols  []string
		shouldError   bool
	}{
		{
			name:          "Get two columns",
			commandStr:    "{name, email}",
			expectedCount: 5,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with limit",
			commandStr:    "{name, email, lim: 1}",
			expectedCount: 1,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with DOWN",
			commandStr:    "{name, email, down: 'name'}",
			expectedCount: 5,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with UP",
			commandStr:    "{name, email, up: 'name'}",
			expectedCount: 5,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with LIKE",
			commandStr:    "{name, email, like: 'user'}",
			expectedCount: 3,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with filter",
			commandStr:    "{name, email, status: 'active'}",
			expectedCount: 3,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with array filter",
			commandStr:    "{name, email, status: ['active', 'pending']}",
			expectedCount: 4,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get three columns with range filter",
			commandStr:    "{id, name, email, id: (1, 3)}",
			expectedCount: 3,
			expectedCols:  []string{"id", "name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with LIM and OFF",
			commandStr:    "{name, email, lim: 2, off: 1}",
			expectedCount: 2,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with LIKE and DOWN",
			commandStr:    "{name, email, like: 'user', down: 'name'}",
			expectedCount: 3,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
		{
			name:          "Get two columns with LIKE and UP",
			commandStr:    "{name, email, like: 'user', up: 'name'}",
			expectedCount: 3,
			expectedCols:  []string{"name", "email"},
			shouldError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args, err := pkg.ParseArg(tc.commandStr)
			if tc.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, "Failed to parse command string: %s", tc.commandStr)

			// Call noqli
			err = pkg.HandleGet(testDB, args, true)
			if tc.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Parse again for validation (to get _columns)
			valArgs, err := pkg.ParseArg(tc.commandStr)
			assert.NoError(t, err)

			// Build validation query using the same logic as noqli
			selectCols := []string{}
			if rawCols, ok := valArgs["_columns"]; ok {
				switch cols := rawCols.(type) {
				case []string:
					selectCols = cols
				case []any:
					for _, c := range cols {
						if s, ok := c.(string); ok {
							selectCols = append(selectCols, s)
						}
					}
				}
			}
			if len(selectCols) == 0 {
				selectCols = tc.expectedCols
			}
			query := "SELECT "
			for i, col := range selectCols {
				if i > 0 {
					query += ", "
				}
				query += "`" + col + "`"
			}
			query += " FROM users"
			var where []string
			var params []any
			for field, value := range valArgs {
				if field == "_columns" || field == "lim" || field == "LIM" || field == "off" || field == "OFF" || field == "like" || field == "LIKE" || field == "up" || field == "UP" || field == "down" || field == "DOWN" {
					continue
				}
				if sliceVal, ok := value.([]any); ok {
					placeholders := make([]string, len(sliceVal))
					for i := range placeholders {
						placeholders[i] = "?"
					}
					where = append(where, "`"+field+"` IN ("+join(placeholders, ",")+")")
					params = append(params, sliceVal...)
				} else if mapVal, ok := value.(map[string]any); ok {
					if rangeSlice, ok := mapVal["range"].([]int); ok && len(rangeSlice) == 2 {
						where = append(where, "`"+field+"` >= ? AND `"+field+"` <= ?")
						params = append(params, rangeSlice[0], rangeSlice[1])
					}
				} else {
					where = append(where, "`"+field+"` = ?")
					params = append(params, value)
				}
			}
			if len(where) > 0 {
				query += " WHERE " + join(where, " AND ")
			}
			// LIKE
			if v, ok := valArgs["like"]; ok {
				likeStr := "%" + v.(string) + "%"
				var likeConds []string
				for _, col := range selectCols {
					likeConds = append(likeConds, "`"+col+"` LIKE ?")
					params = append(params, likeStr)
				}
				if len(where) > 0 {
					query += " AND (" + join(likeConds, " OR ") + ")"
				} else {
					query += " WHERE (" + join(likeConds, " OR ") + ")"
				}
			}
			if v, ok := valArgs["LIKE"]; ok {
				likeStr := "%" + v.(string) + "%"
				var likeConds []string
				for _, col := range selectCols {
					likeConds = append(likeConds, "`"+col+"` LIKE ?")
					params = append(params, likeStr)
				}
				if len(where) > 0 || valArgs["like"] != nil {
					query += " AND (" + join(likeConds, " OR ") + ")"
				} else {
					query += " WHERE (" + join(likeConds, " OR ") + ")"
				}
			}
			// ORDER BY
			if v, ok := valArgs["down"]; ok {
				query += " ORDER BY `" + v.(string) + "` DESC"
			}
			if v, ok := valArgs["up"]; ok {
				query += " ORDER BY `" + v.(string) + "` ASC"
			}
			// LIMIT/OFFSET
			if v, ok := valArgs["lim"]; ok {
				query += " LIMIT ?"
				params = append(params, v)
			}
			if v, ok := valArgs["LIM"]; ok {
				query += " LIMIT ?"
				params = append(params, v)
			}
			if v, ok := valArgs["off"]; ok {
				query += " OFFSET ?"
				params = append(params, v)
			}
			if v, ok := valArgs["OFF"]; ok {
				query += " OFFSET ?"
				params = append(params, v)
			}

			rows, err := testDB.Query(query, params...)
			assert.NoError(t, err, "Error executing validation query")
			defer rows.Close()
			cols, err := rows.Columns()
			assert.NoError(t, err)
			var count int
			var allRows [][]any
			for rows.Next() {
				vals := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for i := range cols {
					ptrs[i] = &vals[i]
				}
				assert.NoError(t, rows.Scan(ptrs...))
				allRows = append(allRows, vals)
				count++
			}
			assert.Equal(t, tc.expectedCount, count, "Expected %d records, got %d for test: %s", tc.expectedCount, count, tc.name)
			assert.Equal(t, len(tc.expectedCols), len(cols), "Expected %d columns, got %d for test: %s", len(tc.expectedCols), len(cols), tc.name)
			for i, col := range cols {
				assert.Equal(t, tc.expectedCols[i], col, "Expected column %s, got %s for test: %s", tc.expectedCols[i], col, tc.name)
			}
		})
	}
}

// join is a helper for joining string slices (no strings.Join to avoid import)
func join(a []string, sep string) string {
	if len(a) == 0 {
		return ""
	}
	res := a[0]
	for i := 1; i < len(a); i++ {
		res += sep + a[i]
	}
	return res
}
