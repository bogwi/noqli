package test

import (
	"fmt"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestCreateCommand(t *testing.T) {
	resetTable(t)

	tests := []struct {
		name     string
		command  string
		args     map[string]any
		expected error
	}{
		{
			name:    "Create Simple User",
			command: "CREATE",
			args: map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			expected: nil,
		},
		{
			name:    "Create User with Multiple Fields",
			command: "CREATE",
			args: map[string]any{
				"name":      "Jane Smith",
				"email":     "jane@example.com",
				"age":       30,
				"active":    true,
				"interests": "reading,coding",
			},
			expected: nil,
		},
		{
			name:     "Create Empty User",
			command:  "CREATE",
			args:     map[string]any{},
			expected: fmt.Errorf("CREATE requires fields to insert"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := pkg.HandleCreate(testDB, tc.args, true)

			if tc.expected == nil {
				assert.NoError(t, err)

				// Verify record was created
				if len(tc.args) > 0 {
					var count int
					err := testDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
					assert.NoError(t, err)
					assert.Greater(t, count, 0)

					// Verify a simple field if present
					if name, ok := tc.args["name"]; ok {
						var dbName string
						err := testDB.QueryRow("SELECT name FROM users WHERE name = ?", name).Scan(&dbName)
						assert.NoError(t, err)
						assert.Equal(t, name, dbName)
					}
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expected.Error())
			}
		})
	}
}
