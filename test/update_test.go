package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

// Use testDB, resetTable, insertTestData from the test package

func TestUpdateCommand(t *testing.T) {
	resetTable(t)
	insertTestData(t)

	// Create a temporary function to override user input for testing
	originalScanln := pkg.ScanForConfirmation
	defer func() {
		pkg.ScanForConfirmation = originalScanln
	}()

	// Mock the ScanForConfirmation function to always return "y" during tests
	pkg.ScanForConfirmation = func() string {
		return "y"
	}

	tests := []struct {
		name          string
		args          map[string]any
		affectedCount int
		filterField   string // Field to filter by when verifying the update
		filterValue   any    // Value to filter by when verifying the update
		updateField   string // Field that was updated
		updateValue   any    // New value that should be set
		shouldError   bool
	}{
		{
			name: "Update Single User by ID",
			args: map[string]any{
				"id":    1,
				"name":  "Updated Name",
				"email": "updated@example.com",
			},
			affectedCount: 1,
			filterField:   "id",
			filterValue:   1,
			updateField:   "name",
			updateValue:   "Updated Name",
			shouldError:   false,
		},
		{
			name: "Update Multiple Users by ID Array",
			args: map[string]any{
				"id":     []any{2, 3},
				"status": "inactive",
			},
			affectedCount: 2,
			filterField:   "id",
			filterValue:   []any{2, 3},
			updateField:   "status",
			updateValue:   "inactive",
			shouldError:   false,
		},
		{
			name: "Update Users in ID Range",
			args: map[string]any{
				"id": map[string]any{
					"range": []int{1, 3},
				},
				"updated": true,
			},
			affectedCount: 3,
			filterField:   "id",
			filterValue:   map[string]any{"range": []int{1, 3}},
			updateField:   "updated",
			updateValue:   true,
			shouldError:   false,
		},
		{
			name: "Update Non-existent User",
			args: map[string]any{
				"id":   999,
				"name": "Won't Update",
			},
			affectedCount: 0,
			filterField:   "id",
			filterValue:   999,
			updateField:   "name",
			updateValue:   "Won't Update",
			shouldError:   true,
		},
		// New test cases for the enhanced filtering functionality
		{
			// This is now testing that we correctly handle filter vs update fields
			// email with a regular string should be an update field, not a filter
			name: "Update All Users with Email Field",
			args: map[string]any{
				"email": "updated@example.com",
			},
			affectedCount: 3, // All records should be updated
			filterField:   "id",
			filterValue:   []any{1, 2, 3},
			updateField:   "email",
			updateValue:   "updated@example.com",
			shouldError:   false,
		},
		{
			name: "Update Users Filtered by Email Array",
			args: map[string]any{
				"email":  []any{"user1@example.com", "user2@example.com"}, // Array = filter
				"status": "batch-updated",                                 // Update field
			},
			affectedCount: 2,
			filterField:   "email",
			filterValue:   []any{"user1@example.com", "user2@example.com"},
			updateField:   "status",
			updateValue:   "batch-updated",
			shouldError:   false,
		},
		{
			name: "Update with Only Filter",
			args: map[string]any{
				"email": []any{"user1@example.com", "user2@example.com"}, // Only a filter, no update fields
			},
			affectedCount: 0,
			filterField:   "email",
			filterValue:   []any{"user1@example.com", "user2@example.com"},
			updateField:   "",
			updateValue:   nil,
			shouldError:   true,
		},
		{
			name: "Update with No Filters (All Records)",
			args: map[string]any{
				"role": "user", // Just an update field
			},
			affectedCount: 3, // All records should be updated
			filterField:   "id",
			filterValue:   []any{1, 2, 3},
			updateField:   "role",
			updateValue:   "user",
			shouldError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetTable(t)
			insertTestData(t)

			err := pkg.HandleUpdate(testDB, tc.args, true)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Construct a query to verify the update
				var query string
				var params []any

				// Build the filter part of the query
				if sliceVal, ok := tc.filterValue.([]any); ok {
					// Handle array values (IN clause)
					placeholders := make([]string, len(sliceVal))
					for i := range placeholders {
						placeholders[i] = "?"
					}
					query = fmt.Sprintf("SELECT COUNT(*) FROM users WHERE `%s` IN (%s)",
						tc.filterField, strings.Join(placeholders, ","))

					params = append(params, sliceVal...)
				} else if mapVal, ok := tc.filterValue.(map[string]any); ok {
					// Handle range queries
					if rangeSlice, ok := mapVal["range"].([]int); ok && len(rangeSlice) == 2 {
						query = fmt.Sprintf("SELECT COUNT(*) FROM users WHERE `%s` >= ? AND `%s` <= ?",
							tc.filterField, tc.filterField)
						params = append(params, rangeSlice[0], rangeSlice[1])
					}
				} else {
					// Handle simple value
					query = fmt.Sprintf("SELECT COUNT(*) FROM users WHERE `%s` = ?",
						tc.filterField)
					params = append(params, tc.filterValue)
				}

				// Execute the verification query
				var count int
				err := testDB.QueryRow(query, params...).Scan(&count)

				// If we don't expect any affected rows, that's fine
				if tc.affectedCount == 0 {
					// Skip verification as we expect no records
				} else {
					assert.NoError(t, err, "Error executing validation query")
					assert.Equal(t, tc.affectedCount, count,
						"Expected %d records for filter, got %d for test: %s",
						tc.affectedCount, count, tc.name)

					// For each record that matches the filter, check if update was applied
					// But we need to check in a separate query to avoid counting errors
					if count > 0 {
						// Verify the update was applied to the records
						updateQuery := query + fmt.Sprintf(" AND `%s` = ?", tc.updateField)
						updateParams := append([]any{}, params...)
						updateParams = append(updateParams, tc.updateValue)

						var updatedCount int
						err = testDB.QueryRow(updateQuery, updateParams...).Scan(&updatedCount)
						assert.NoError(t, err, "Error executing update validation query")
						assert.Equal(t, count, updatedCount,
							"Not all matching records were updated in test: %s", tc.name)
					}
				}
			}
		})
	}
}
