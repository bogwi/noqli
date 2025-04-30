package test

import (
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestDeleteCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		expectedIDs []int // IDs that should remain after deletion
		shouldError bool
	}{
		{
			name: "Delete Single User",
			args: map[string]any{
				"id": 1,
			},
			expectedIDs: []int{2, 3},
			shouldError: false,
		},
		{
			name: "Delete Multiple Users",
			args: map[string]any{
				"id": []any{1, 2},
			},
			expectedIDs: []int{3},
			shouldError: false,
		},
		{
			name: "Delete Users in Range",
			args: map[string]any{
				"id": map[string]any{
					"range": []int{1, 2},
				},
			},
			expectedIDs: []int{3},
			shouldError: false,
		},
		{
			name: "Delete Non-existent User",
			args: map[string]any{
				"id": 999,
			},
			expectedIDs: []int{1, 2, 3},
			shouldError: true,
		},
		{
			name: "Delete Without ID",
			args: map[string]any{
				"name": "Missing ID",
			},
			expectedIDs: []int{1, 2, 3},
			shouldError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetTable(t)
			insertTestData(t)

			err := pkg.HandleDelete(testDB, tc.args, true)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify expected IDs still exist
			for _, id := range tc.expectedIDs {
				var count int
				err := testDB.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", id).Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count, "ID %d should exist after deletion", id)
			}

			// Verify total count matches expected remaining IDs
			var totalCount int
			err = testDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalCount)
			assert.NoError(t, err)
			assert.Equal(t, len(tc.expectedIDs), totalCount, "Unexpected number of records after deletion")
		})
	}
}
