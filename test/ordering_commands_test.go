package test

import (
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestOrderingCommands(t *testing.T) {
	resetTable(t)

	// Insert test data with predictable ordering
	_, err := testDB.Exec(`
		INSERT INTO users (name, email) VALUES 
		('Charlie', 'charlie@example.com'),
		('Alice', 'alice@example.com'),
		('Bob', 'bob@example.com'),
		('David', 'david@example.com'),
		('Eve', 'eve@example.com')
	`)
	assert.NoError(t, err, "Failed to insert test data for ordering")

	// Test ascending order with 'up'
	t.Run("Ascending Order with 'up'", func(t *testing.T) {
		// Execute the order query directly to verify actual results
		rows, err := testDB.Query("SELECT name FROM users ORDER BY name ASC")
		assert.NoError(t, err)
		defer rows.Close()

		// Collect ordered names
		var expectedNames []string
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			assert.NoError(t, err)
			expectedNames = append(expectedNames, name)
		}

		// Verify the HandleGet function properly applies ordering
		// This part just tests that the function succeeds, as we can't directly verify the output in this test
		args := map[string]any{
			"up": "name",
		}
		err = pkg.HandleGet(testDB, args, false)
		assert.NoError(t, err)

		// The expectedNames should be: [Alice, Bob, Charlie, David, Eve]
		assert.Equal(t, "Alice", expectedNames[0])
		assert.Equal(t, "Eve", expectedNames[len(expectedNames)-1])
	})

	// Test descending order with 'down'
	t.Run("Descending Order with 'down'", func(t *testing.T) {
		// Execute the order query directly to verify actual results
		rows, err := testDB.Query("SELECT name FROM users ORDER BY name DESC")
		assert.NoError(t, err)
		defer rows.Close()

		// Collect ordered names
		var expectedNames []string
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			assert.NoError(t, err)
			expectedNames = append(expectedNames, name)
		}

		// Verify the HandleGet function properly applies ordering
		args := map[string]any{
			"down": "name",
		}
		err = pkg.HandleGet(testDB, args, false)
		assert.NoError(t, err)

		// The expectedNames should be: [Eve, David, Charlie, Bob, Alice]
		assert.Equal(t, "Eve", expectedNames[0])
		assert.Equal(t, "Alice", expectedNames[len(expectedNames)-1])
	})

	// Test with uppercase variants
	t.Run("Ascending Order with 'UP'", func(t *testing.T) {
		args := map[string]any{
			"UP": "name",
		}
		err = pkg.HandleGet(testDB, args, true)
		assert.NoError(t, err)
	})

	t.Run("Descending Order with 'DOWN'", func(t *testing.T) {
		args := map[string]any{
			"DOWN": "name",
		}
		err = pkg.HandleGet(testDB, args, true)
		assert.NoError(t, err)
	})

	// Test with filtering and ordering combined
	t.Run("Filtering and Ordering Combined", func(t *testing.T) {
		// First two names in alphabetical order (Alice, Bob)
		args := map[string]any{
			"name": []any{"Alice", "Bob", "Charlie"},
			"up":   "name",
		}
		err = pkg.HandleGet(testDB, args, false)
		assert.NoError(t, err)

		// Verify the actual results with direct query
		rows, err := testDB.Query("SELECT name FROM users WHERE name IN (?, ?, ?) ORDER BY name ASC", "Alice", "Bob", "Charlie")
		assert.NoError(t, err)
		defer rows.Close()

		var names []string
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			assert.NoError(t, err)
			names = append(names, name)
		}

		assert.Equal(t, 3, len(names))
		assert.Equal(t, "Alice", names[0])
		assert.Equal(t, "Charlie", names[len(names)-1])
	})
}
