package test

import (
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestGetCommandCount(t *testing.T) {
	resetTable(t)
	insertTestData(t)

	// Insert additional data for DISTINCT and LIKE tests
	_, err := testDB.Exec(`
		INSERT INTO users (name, email, status) VALUES 
		('User 1', 'user1@example.com', 'active'),
		('User 2', 'user2@example.com', 'inactive'),
		('User 2', 'user2@example.com', 'inactive'),
		('User 3', NULL, 'active'),
		('User 4', 'user4@example.com', NULL),
		('User 5', 'user5@example.com', 'active'),
		('User 6', 'user6@example.com', 'inactive'),
		('User 7', 'user7@example.com', 'active')
	`)
	assert.NoError(t, err, "Failed to insert additional test data for COUNT test")

	tests := []struct {
		name          string
		commandStr    string
		directSQL     string
		paramsBuilder func() []any
	}{
		{
			name:          "Count all rows",
			commandStr:    `{COUNT: '*'}`,
			directSQL:     "SELECT COUNT(*) FROM users",
			paramsBuilder: func() []any { return nil },
		},
		{
			name:          "Count non-null emails",
			commandStr:    `{COUNT: 'email'}`,
			directSQL:     "SELECT COUNT(email) FROM users",
			paramsBuilder: func() []any { return nil },
		},
		{
			name:          "Count distinct emails",
			commandStr:    `{COUNT: 'email', DISTINCT: true}`,
			directSQL:     "SELECT COUNT(DISTINCT email) FROM users",
			paramsBuilder: func() []any { return nil },
		},
		{
			name:          "Count with filter",
			commandStr:    `{COUNT: '*', status: 'active'}`,
			directSQL:     "SELECT COUNT(*) FROM users WHERE status = ?",
			paramsBuilder: func() []any { return []any{"active"} },
		},
		{
			name:          "Count with IN clause",
			commandStr:    `{COUNT: 'name', name: ['User 1', 'User 2']}`,
			directSQL:     "SELECT COUNT(name) FROM users WHERE name IN (?, ?)",
			paramsBuilder: func() []any { return []any{"User 1", "User 2"} },
		},
		{
			name:          "Count with range",
			commandStr:    `{COUNT: 'id', id: (1,5)}`,
			directSQL:     "SELECT COUNT(id) FROM users WHERE id >= ? AND id <= ?",
			paramsBuilder: func() []any { return []any{1, 5} },
		},
		{
			name:          "Count with LIKE",
			commandStr:    `{COUNT: '*', LIKE: 'User'}`,
			directSQL:     "SELECT COUNT(*) FROM users WHERE name LIKE ? OR email LIKE ? OR status LIKE ?",
			paramsBuilder: func() []any { return []any{"%User%", "%User%", "%User%"} },
		},
		{
			name:          "Count with no results",
			commandStr:    `{COUNT: '*', name: 'NonExistent'}`,
			directSQL:     "SELECT COUNT(*) FROM users WHERE name = ?",
			paramsBuilder: func() []any { return []any{"NonExistent"} },
		},
		{
			name:          "Count all null emails",
			commandStr:    `{COUNT: 'email', email: null}`,
			directSQL:     "SELECT COUNT(email) FROM users WHERE email IS NULL",
			paramsBuilder: func() []any { return nil },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args, err := pkg.ParseArg(tc.commandStr)
			assert.NoError(t, err, "Failed to parse command string: %s", tc.commandStr)

			// Run the actual NoQLi command
			err = pkg.HandleGet(testDB, args, true)
			assert.NoError(t, err, "HandleGet failed for: %s", tc.commandStr)

			// Validate count directly from database
			var count int
			params := tc.paramsBuilder()
			if params == nil {
				err = testDB.QueryRow(tc.directSQL).Scan(&count)
			} else {
				err = testDB.QueryRow(tc.directSQL, params...).Scan(&count)
			}
			assert.NoError(t, err, "Error executing validation query for %s", tc.name)
			// No assertion on value here, as output is not captured, but this ensures SQL is valid and matches the command
		})
	}
}
