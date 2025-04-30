package test

import (
	"fmt"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestComprehensiveUpdateOperations(t *testing.T) {
	resetTable(t)

	// Insert more diverse test data for thorough testing
	_, err := testDB.Exec(`
		INSERT INTO users (name, email, status, category, priority, tags) VALUES 
		('User 1', 'user1@example.com', 'active', 'customer', 'high', 'premium,support'),
		('User 2', 'user2@example.com', 'inactive', 'vendor', 'medium', 'basic'),
		('User 3', 'user3@example.com', 'pending', 'customer', 'low', 'trial'),
		('User 4', 'user4@example.com', 'active', 'admin', 'high', 'internal,premium'),
		('User 5', 'user5@example.com', 'pending', 'vendor', 'medium', 'basic,support')
	`)
	assert.NoError(t, err, "Failed to insert comprehensive test data")

	// Override user confirmation to always return "y" for tests
	originalScanln := pkg.ScanForConfirmation
	defer func() {
		pkg.ScanForConfirmation = originalScanln
	}()
	pkg.ScanForConfirmation = func() string {
		return "y"
	}

	// Helper function to verify updates
	verifyUpdate := func(filterQuery string, filterParams []any, expectedCount int,
		updateField string, updateValue any) {

		// First verify the count of records matching the filter
		var count int
		err := testDB.QueryRow(filterQuery, filterParams...).Scan(&count)
		assert.NoError(t, err, "Error executing count query")
		assert.Equal(t, expectedCount, count, "Expected %d filtered records, got %d", expectedCount, count)

		if expectedCount > 0 && updateField != "" {
			// Verify the update was applied correctly
			updateQuery := filterQuery + fmt.Sprintf(" AND `%s` = ?", updateField)
			updateParams := append([]any{}, filterParams...)
			updateParams = append(updateParams, updateValue)

			var updatedCount int
			err = testDB.QueryRow(updateQuery, updateParams...).Scan(&updatedCount)
			assert.NoError(t, err, "Error executing update verification query")
			assert.Equal(t, expectedCount, updatedCount, "Not all records were updated correctly")
		}
	}

	// 1. Update a single record by ID
	t.Run("Update Single Record by ID", func(t *testing.T) {
		args := map[string]any{
			"id":     1,
			"status": "updated-status",
			"notes":  "New field with dynamic column creation",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE id = ?",
			[]any{1},
			1,
			"status",
			"updated-status",
		)

		// Verify the new column was created
		var notes string
		err = testDB.QueryRow("SELECT notes FROM users WHERE id = ?", 1).Scan(&notes)
		assert.NoError(t, err)
		assert.Equal(t, "New field with dynamic column creation", notes)
	})

	// 2. Update multiple records by ID array
	t.Run("Update Multiple Records by ID Array", func(t *testing.T) {
		args := map[string]any{
			"id":       []any{2, 3, 4},
			"category": "batch-updated",
			"modified": true,
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE id IN (?, ?, ?)",
			[]any{2, 3, 4},
			3,
			"category",
			"batch-updated",
		)
	})

	// 3. Update records in ID range
	t.Run("Update Records in ID Range", func(t *testing.T) {
		args := map[string]any{
			"id": map[string]any{
				"range": []int{3, 5},
			},
			"range_updated": "yes",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE id >= ? AND id <= ?",
			[]any{3, 5},
			3,
			"range_updated",
			"yes",
		)
	})

	// 4. Update records filtered by any column
	t.Run("Update Records Filtered by Any Column", func(t *testing.T) {
		// In NoQLi, email with a single string will be treated as an update field, not a filter field
		// We need to reset the data first to ensure we have a fresh state
		resetTable(t)
		_, err := testDB.Exec(`
			INSERT INTO users (name, email, status, category, priority, tags) VALUES 
			('User 1', 'user1@example.com', 'active', 'customer', 'high', 'premium,support'),
			('User 2', 'user2@example.com', 'inactive', 'vendor', 'medium', 'basic'),
			('User 3', 'user3@example.com', 'pending', 'customer', 'low', 'trial'),
			('User 4', 'user4@example.com', 'active', 'admin', 'high', 'internal,premium'),
			('User 5', 'user5@example.com', 'pending', 'vendor', 'medium', 'basic,support')
		`)
		assert.NoError(t, err)

		// Create a test with a field that will be treated as a filter (using id)
		args := map[string]any{
			"id":     5, // Use ID as filter
			"status": "approved",
			"email":  "filtered.user@example.com", // This will be an update field, not a filter
		}

		err = pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		// Verify the update using ID as the filter
		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE id = ?",
			[]any{5},
			1,
			"status",
			"approved",
		)

		// Also verify that email was updated
		var email string
		err = testDB.QueryRow("SELECT email FROM users WHERE id = ?", 5).Scan(&email)
		assert.NoError(t, err)
		assert.Equal(t, "filtered.user@example.com", email, "Email should be updated for user with ID 5")
	})

	// 5. Update records filtered by array values
	t.Run("Update Records Filtered by Array Values", func(t *testing.T) {
		// First reset the data to ensure we have a controlled state
		resetTable(t)
		_, err := testDB.Exec(`
			INSERT INTO users (name, email, status, category, priority, tags) VALUES 
			('User 1', 'user1@example.com', 'active', 'customer', 'high', 'premium,support'),
			('User 2', 'user2@example.com', 'inactive', 'vendor', 'medium', 'basic'),
			('User 3', 'user3@example.com', 'pending', 'customer', 'low', 'trial'),
			('User 4', 'user4@example.com', 'active', 'admin', 'high', 'internal,premium'),
			('User 5', 'user5@example.com', 'pending', 'vendor', 'medium', 'basic,support')
		`)
		assert.NoError(t, err)

		args := map[string]any{
			"status":      []any{"pending", "active"},
			"bulk_update": "processed",
		}

		err = pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		// Query the database directly to see exactly which records were matched and updated
		rows, err := testDB.Query("SELECT id, status, bulk_update FROM users WHERE bulk_update = 'processed'")
		assert.NoError(t, err)
		defer rows.Close()

		var updatedRecords []int
		for rows.Next() {
			var id int
			var status, bulkUpdate string
			err := rows.Scan(&id, &status, &bulkUpdate)
			assert.NoError(t, err)
			updatedRecords = append(updatedRecords, id)
		}

		// Based on NoQLi's actual behavior, 4 records are matched:
		// ID 1 (active), ID 3 (pending), ID 4 (active), ID 5 (pending)
		assert.Equal(t, 4, len(updatedRecords), "Should update 4 records with status 'active' or 'pending'")
		assert.Contains(t, updatedRecords, 1)
		assert.Contains(t, updatedRecords, 3)
		assert.Contains(t, updatedRecords, 4)
		assert.Contains(t, updatedRecords, 5)

		// Also verify one of the records using our helper function to ensure the field was updated correctly
		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE status = 'active'",
			[]any{},
			2, // 2 active records (ID 1 and 4)
			"bulk_update",
			"processed",
		)
	})

	// 6. Update all records (with confirmation)
	t.Run("Update All Records", func(t *testing.T) {
		args := map[string]any{
			"global_field": "applied-to-all",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		// Verify all records were updated
		var count int
		err = testDB.QueryRow("SELECT COUNT(*) FROM users WHERE global_field = ?", "applied-to-all").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 5, count, "All records should have been updated")
	})

	// Test error conditions and edge cases

	// 7. Update with non-existent ID
	t.Run("Update Non-existent ID", func(t *testing.T) {
		args := map[string]any{
			"id":    999,
			"field": "value",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no records matched")
	})

	// 8. Update with only filter fields (no update fields)
	t.Run("Update with Filter-only", func(t *testing.T) {
		args := map[string]any{
			"status": []any{"active", "pending"},
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires fields to update")
	})

	// 9. Update with empty args
	t.Run("Update with Empty Args", func(t *testing.T) {
		args := map[string]any{}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.Error(t, err)
	})

	// 10. Update with invalid range
	t.Run("Update with Invalid Range", func(t *testing.T) {
		args := map[string]any{
			"id": map[string]any{
				"range": []int{10, 5}, // Invalid: start > end
			},
			"field": "value",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.Error(t, err) // Should error because no records match
	})

	// 11. Update with multiple field types
	t.Run("Update with Multiple Field Types", func(t *testing.T) {
		args := map[string]any{
			"id":            1,
			"status":        "complex-update",
			"numeric_value": 42,
			"boolean_value": true,
			"nullish":       nil,
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		// Verify different field types were updated correctly
		var status string
		var numericValue int
		var booleanValue bool

		err = testDB.QueryRow("SELECT status, numeric_value, boolean_value FROM users WHERE id = ?", 1).Scan(&status, &numericValue, &booleanValue)
		assert.NoError(t, err)
		assert.Equal(t, "complex-update", status)
		assert.Equal(t, 42, numericValue)
		assert.Equal(t, true, booleanValue)
	})

	// 12. Mixed update and filter with same field name but different value types
	t.Run("Mixed Update and Filter with Same Field", func(t *testing.T) {
		// Setup: Set different categories
		_, err := testDB.Exec("UPDATE users SET category='type-A' WHERE id IN (1, 2)")
		assert.NoError(t, err)
		_, err = testDB.Exec("UPDATE users SET category='type-B' WHERE id IN (3, 4, 5)")
		assert.NoError(t, err)

		// Use category as both filter (array) and update (string)
		args := map[string]any{
			"category":   []any{"type-A"}, // This is a filter
			"new_status": "special",       // This is an update field
		}

		err = pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		// Verify only type-A records were updated
		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE category = ?",
			[]any{"type-A"},
			2,
			"new_status",
			"special",
		)

		// Verify type-B records were not updated
		var count int
		err = testDB.QueryRow("SELECT COUNT(*) FROM users WHERE category = ? AND new_status = ?",
			"type-B", "special").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count, "Records with category type-B should not be updated")
	})

	// 13. Complex test with multiple mixed filters and multiple field updates
	t.Run("Complex Multiple Filters and Updates", func(t *testing.T) {
		// Reset data to a known state
		resetTable(t)
		_, err := testDB.Exec(`
			INSERT INTO users (name, email, status, category, priority, tags) VALUES 
			('User 1', 'user1@example.com', 'active', 'customer', 'high', 'premium,support'),
			('User 2', 'user2@example.com', 'inactive', 'vendor', 'medium', 'basic'),
			('User 3', 'user3@example.com', 'pending', 'customer', 'low', 'trial'),
			('User 4', 'user4@example.com', 'active', 'admin', 'high', 'internal,premium'),
			('User 5', 'user5@example.com', 'pending', 'vendor', 'medium', 'basic,support')
		`)
		assert.NoError(t, err)

		// Complex update with multiple filters and multiple update fields
		args := map[string]any{
			// Filter fields (using array notation)
			"status":   []any{"active", "pending"},
			"priority": []any{"high"},

			// Update fields
			"processed":  true,
			"level":      "advanced",
			"updated_at": "2023-08-15",
			"score":      95.5,
		}

		err = pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		// This should match users 1 and 4 (active+high priority)
		// Construct complex WHERE clause for verification
		verifyQuery := `
			SELECT COUNT(*) FROM users 
			WHERE status IN (?, ?) 
			AND priority IN (?)
		`
		verifyParams := []any{"active", "pending", "high"}

		// Verify matched records
		var matchedCount int
		err = testDB.QueryRow(verifyQuery, verifyParams...).Scan(&matchedCount)
		assert.NoError(t, err)
		assert.Equal(t, 2, matchedCount, "Should match 2 records with status in (active,pending) AND priority=high")

		// Verify all fields were updated correctly on matching records
		verifyUpdatesQuery := verifyQuery + ` 
			AND processed = ? 
			AND level = ? 
			AND updated_at = ?
			AND score = ?
		`
		updateVerifyParams := append(verifyParams, true, "advanced", "2023-08-15", 95.5)

		var updatedCount int
		err = testDB.QueryRow(verifyUpdatesQuery, updateVerifyParams...).Scan(&updatedCount)
		assert.NoError(t, err)
		assert.Equal(t, 2, updatedCount, "All matching records should have all fields updated correctly")

		// Verify non-matching records were not updated
		var nonUpdatedCount int
		err = testDB.QueryRow("SELECT COUNT(*) FROM users WHERE processed IS NULL").Scan(&nonUpdatedCount)
		assert.NoError(t, err)
		assert.Equal(t, 3, nonUpdatedCount, "Non-matching records should not be updated")
	})
}
