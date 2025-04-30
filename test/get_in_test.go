package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestGetCommandIN(t *testing.T) {
	resetTable(t)

	// Mock scanForConfirmation to always return "y"
	oldScanForConfirmation := pkg.ScanForConfirmation
	pkg.ScanForConfirmation = func() string {
		return "y"
	}
	defer func() {
		pkg.ScanForConfirmation = oldScanForConfirmation
	}()

	// Insert test data with specific names for testing IN clause
	_, err := testDB.Exec(`
		INSERT INTO users (name, email, status) VALUES 
		('XXX', 'alice@example.com', 'active'),
		('Y', 'actve@c.bcom', 'clean'),
		('Y', 'actve@c.bcom', 'clean'),
		(NULL, NULL, NULL)
	`)
	assert.NoError(t, err, "Failed to insert test data for IN test")

	// Add a phone column - without using IF NOT EXISTS which is not supported in all MySQL versions
	_, err = testDB.Exec(`
		ALTER TABLE users ADD COLUMN phone VARCHAR(255) DEFAULT NULL
	`)
	assert.NoError(t, err, "Failed to add phone column")

	_, err = testDB.Exec(`
		UPDATE users SET phone = '0' WHERE id = 4
	`)
	assert.NoError(t, err, "Failed to update phone")

	// First verify we can retrieve all records
	err = pkg.HandleGet(testDB, nil, true)
	assert.NoError(t, err, "Failed to get all records")

	// Test the IN clause with string values using actual command strings
	tests := []struct {
		name          string
		commandStr    string
		expectedCount int
		shouldError   bool
	}{
		{
			name:          "Get users by name IN clause",
			commandStr:    `{name: ["XXX", "Y"]}`,
			expectedCount: 3, // Should find 3 records (XXX and two Y's)
			shouldError:   false,
		},
		{
			name:          "Get users by status IN clause",
			commandStr:    `{status: ["active", "clean"]}`,
			expectedCount: 3, // Should find 3 records with status 'active' or 'clean'
			shouldError:   false,
		},
		{
			name:          "Get users by non-existent values",
			commandStr:    `{name: ["NonExistent1", "NonExistent2"]}`,
			expectedCount: 0,
			shouldError:   false,
		},
		{
			name:          "Get users with mixed data types in array",
			commandStr:    `{name: ["XXX", 123]}`,
			expectedCount: 1, // Should only find 'XXX'
			shouldError:   false,
		},
		{
			// Test with single quotes
			name:          "Get users with single quotes in array",
			commandStr:    `{name: ['XXX', 'Y']}`,
			expectedCount: 3,
			shouldError:   false,
		},
		{
			// Test with no quotes (should still work for strings)
			name:          "Get users with unquoted values in array",
			commandStr:    `{name: [XXX, Y]}`,
			expectedCount: 3,
			shouldError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the command string through the actual parser
			args, err := pkg.ParseArg(tc.commandStr)
			if tc.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, "Failed to parse command string: %s", tc.commandStr)

			// Log the parsed args for debugging
			t.Logf("Parsed args: %+v", args)

			// Execute the noqli command with the parsed args
			err = pkg.HandleGet(testDB, args, true)
			if tc.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Validate count directly from database
			if args != nil {
				query := "SELECT COUNT(*) FROM users WHERE 1=1"
				var params []any

				// Add conditions for each argument
				for field, value := range args {
					if sliceVal, ok := value.([]any); ok {
						// Handle array values (IN clause)
						placeholders := make([]string, len(sliceVal))
						for i := range placeholders {
							placeholders[i] = "?"
						}
						query += fmt.Sprintf(" AND `%s` IN (%s)", field, strings.Join(placeholders, ","))
						params = append(params, sliceVal...)
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
