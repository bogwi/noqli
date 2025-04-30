package test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestHandleCommand(t *testing.T) {
	resetTable(t)

	tests := []struct {
		name       string
		input      string
		shouldPass bool
	}{
		{
			name:       "Valid CREATE",
			input:      "CREATE {name: 'Test User', email: 'test@example.com'}",
			shouldPass: true,
		},
		{
			name:       "Valid GET All",
			input:      "GET",
			shouldPass: true,
		},
		{
			name:       "Valid GET By ID",
			input:      "GET 1",
			shouldPass: true,
		},
		{
			name:       "Valid UPDATE",
			input:      "UPDATE {id: 1, name: 'Updated Name'}",
			shouldPass: true,
		},
		{
			name:       "Valid DELETE",
			input:      "DELETE {id: 1}",
			shouldPass: true,
		},
		{
			name:       "Invalid Command",
			input:      "INVALID {id: 1}",
			shouldPass: false,
		},
		{
			name:       "Invalid Syntax",
			input:      "CREATE invalid",
			shouldPass: false,
		},
	}

	// First create a test user for update/delete operations
	err := pkg.HandleCreate(testDB, map[string]any{
		"name":  "Test User",
		"email": "test@example.com",
	}, true)
	assert.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// We need to recreate handleCommand here for testing
			err := func(db *sql.DB, line string) error {
				trimmed := strings.TrimSpace(line)

				// Parse command using regex
				re := pkg.GetCommandRegex()
				matches := re.FindStringSubmatch(trimmed)

				if matches == nil {
					return fmt.Errorf("invalid command. Use CREATE, GET, UPDATE, DELETE, or EXIT")
				}

				command := strings.ToUpper(matches[1])
				argStr := matches[2]

				var argObj map[string]any
				var err error

				if argStr != "" {
					argObj, err = pkg.ParseArg(argStr)
					if err != nil {
						return fmt.Errorf("could not parse argument object: %v", err)
					}
				}

				switch command {
				case "CREATE":
					return pkg.HandleCreate(db, argObj, true)
				case "GET":
					return pkg.HandleGet(db, argObj, true)
				case "UPDATE":
					return pkg.HandleUpdate(db, argObj, true)
				case "DELETE":
					return pkg.HandleDelete(db, argObj, true)
				default:
					return fmt.Errorf("unknown command: %s", command)
				}
			}(testDB, tc.input)

			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
