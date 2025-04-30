package test

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/bogwi/noqli/pkg"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

const (
	testDBName = "noqli_test_db"
	testTable  = "users"
)

var (
	testDB     *sql.DB
	mainDB     *sql.DB
	testDBHost string
	testDBUser string
	testDBPass string
)

func TestMain(m *testing.M) {
	// Load test environment variables
	loadTestEnv()

	// Setup test database
	if err := setupTestDatabase(); err != nil {
		fmt.Printf("Failed to set up test database: %v\n", err)
		os.Exit(1)
	}

	// Set the CurrentDB and CurrentTable for testing
	pkg.CurrentDB = testDBName
	pkg.CurrentTable = testTable

	// Run tests
	exitCode := m.Run()

	// Cleanup
	if err := cleanupTestDatabase(); err != nil {
		fmt.Printf("Failed to clean up test database: %v\n", err)
	}

	os.Exit(exitCode)
}

func loadTestEnv() {
	// Try to load from .env.test first, fall back to defaults if not present
	if err := godotenv.Load(".env.test"); err != nil {
		fmt.Println("No .env.test file found, using default test credentials")
	}

	// Set test database connection parameters with defaults
	testDBHost = getEnvOrDefault("TEST_DB_HOST", "localhost")
	testDBUser = getEnvOrDefault("TEST_DB_USER", "root")
	testDBPass = getEnvOrDefault("TEST_DB_PASS", "1234")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupTestDatabase() error {
	// Connect to MySQL server (without database)
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/", testDBUser, testDBPass, testDBHost)
	var err error
	mainDB, err = sql.Open("mysql", connStr)
	if err != nil {
		return fmt.Errorf("error connecting to MySQL: %v", err)
	}

	// Test connection
	if err := mainDB.Ping(); err != nil {
		return fmt.Errorf("error connecting to MySQL: %v", err)
	}

	// Drop test database if it exists
	_, err = mainDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
	if err != nil {
		return fmt.Errorf("error dropping test database: %v", err)
	}

	// Create test database
	_, err = mainDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		return fmt.Errorf("error creating test database: %v", err)
	}

	// Connect to the test database
	testConnStr := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		testDBUser, testDBPass, testDBHost, testDBName)
	testDB, err = sql.Open("mysql", testConnStr)
	if err != nil {
		return fmt.Errorf("error connecting to test database: %v", err)
	}

	// Create test table
	_, err = testDB.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255)
		)
	`, testTable))
	if err != nil {
		return fmt.Errorf("error creating test table: %v", err)
	}

	// Override the table name for testing
	originalTableName := "users"
	_, err = testDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", originalTableName))
	if err != nil {
		return fmt.Errorf("error dropping original table: %v", err)
	}

	// Create the users table expected by the application
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255)
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating users table: %v", err)
	}

	return nil
}

func cleanupTestDatabase() error {
	// Close database connections
	if testDB != nil {
		testDB.Close()
	}

	// Drop the test database
	if mainDB != nil {
		_, err := mainDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
		mainDB.Close()
		if err != nil {
			return fmt.Errorf("error dropping test database: %v", err)
		}
	}

	return nil
}

// Helper function to reset table between tests
func resetTable(t *testing.T) {
	_, err := testDB.Exec("TRUNCATE TABLE users")
	assert.NoError(t, err, "Failed to truncate users table")
}

// Helper function to insert test data
func insertTestData(t *testing.T) {
	_, err := testDB.Exec(`
		INSERT INTO users (name, email) VALUES 
		('User 1', 'user1@example.com'),
		('User 2', 'user2@example.com'),
		('User 3', 'user3@example.com')
	`)
	assert.NoError(t, err, "Failed to insert test data")
}

// Test CREATE command
func TestCreateCommand(t *testing.T) {
	resetTable(t)

	tests := []struct {
		name     string
		command  string
		args     map[string]interface{}
		expected error
	}{
		{
			name:    "Create Simple User",
			command: "CREATE",
			args: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			expected: nil,
		},
		{
			name:    "Create User with Multiple Fields",
			command: "CREATE",
			args: map[string]interface{}{
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
			args:     map[string]interface{}{},
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

// Test GET command
func TestGetCommand(t *testing.T) {
	resetTable(t)
	insertTestData(t)

	tests := []struct {
		name          string
		args          map[string]interface{}
		expectedCount int
		shouldError   bool
	}{
		{
			name:          "Get All Users",
			args:          nil,
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name: "Get User by ID",
			args: map[string]interface{}{
				"id": 1,
			},
			expectedCount: 1,
			shouldError:   false,
		},
		{
			name: "Get Multiple Users by ID",
			args: map[string]interface{}{
				"id": []interface{}{1, 2},
			},
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name: "Get Users by ID Range",
			args: map[string]interface{}{
				"id": map[string]interface{}{
					"range": []int{1, 3},
				},
			},
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name: "Get Non-existent User",
			args: map[string]interface{}{
				"id": 999,
			},
			expectedCount: 0,
			shouldError:   false, // NoQLi doesn't error on no records, just returns "No records found"
		},
		// New test cases for filtering by non-ID columns
		{
			name: "Get User by Email",
			args: map[string]interface{}{
				"email": "user1@example.com",
			},
			expectedCount: 1,
			shouldError:   false,
		},
		{
			name: "Get Multiple Users by Email Array",
			args: map[string]interface{}{
				"email": []interface{}{"user1@example.com", "user2@example.com"},
			},
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name: "Get Users by Multiple Criteria",
			args: map[string]interface{}{
				"name":  "User 1",
				"email": "user1@example.com",
			},
			expectedCount: 1,
			shouldError:   false,
		},
		{
			name: "Get Users by Non-existent Email",
			args: map[string]interface{}{
				"email": "nonexistent@example.com",
			},
			expectedCount: 0,
			shouldError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := pkg.HandleGet(testDB, tc.args, true)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Validate the query results directly from the database
			if tc.args != nil {
				query := "SELECT COUNT(*) FROM users WHERE 1=1"
				var params []interface{}

				// Add conditions for each argument
				for field, value := range tc.args {
					if sliceVal, ok := value.([]interface{}); ok {
						// Handle array values (IN clause)
						placeholders := make([]string, len(sliceVal))
						for i := range placeholders {
							placeholders[i] = "?"
						}
						query += fmt.Sprintf(" AND `%s` IN (%s)", field, strings.Join(placeholders, ","))
						for _, v := range sliceVal {
							params = append(params, v)
						}
					} else if mapVal, ok := value.(map[string]interface{}); ok {
						// Handle range queries
						if rangeSlice, ok := mapVal["range"].([]int); ok && len(rangeSlice) == 2 {
							query += fmt.Sprintf(" AND `%s` >= ? AND `%s` <= ?", field, field)
							params = append(params, rangeSlice[0], rangeSlice[1])
						}
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

// Test UPDATE command
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
		args          map[string]interface{}
		affectedCount int
		filterField   string      // Field to filter by when verifying the update
		filterValue   interface{} // Value to filter by when verifying the update
		updateField   string      // Field that was updated
		updateValue   interface{} // New value that should be set
		shouldError   bool
	}{
		{
			name: "Update Single User by ID",
			args: map[string]interface{}{
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
			args: map[string]interface{}{
				"id":     []interface{}{2, 3},
				"status": "inactive",
			},
			affectedCount: 2,
			filterField:   "id",
			filterValue:   []interface{}{2, 3},
			updateField:   "status",
			updateValue:   "inactive",
			shouldError:   false,
		},
		{
			name: "Update Users in ID Range",
			args: map[string]interface{}{
				"id": map[string]interface{}{
					"range": []int{1, 3},
				},
				"updated": true,
			},
			affectedCount: 3,
			filterField:   "id",
			filterValue:   map[string]interface{}{"range": []int{1, 3}},
			updateField:   "updated",
			updateValue:   true,
			shouldError:   false,
		},
		{
			name: "Update Non-existent User",
			args: map[string]interface{}{
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
			args: map[string]interface{}{
				"email": "updated@example.com",
			},
			affectedCount: 3, // All records should be updated
			filterField:   "id",
			filterValue:   []interface{}{1, 2, 3},
			updateField:   "email",
			updateValue:   "updated@example.com",
			shouldError:   false,
		},
		{
			name: "Update Users Filtered by Email Array",
			args: map[string]interface{}{
				"email":  []interface{}{"user1@example.com", "user2@example.com"}, // Array = filter
				"status": "batch-updated",                                         // Update field
			},
			affectedCount: 2,
			filterField:   "email",
			filterValue:   []interface{}{"user1@example.com", "user2@example.com"},
			updateField:   "status",
			updateValue:   "batch-updated",
			shouldError:   false,
		},
		{
			name: "Update with Only Filter",
			args: map[string]interface{}{
				"email": []interface{}{"user1@example.com", "user2@example.com"}, // Only a filter, no update fields
			},
			affectedCount: 0,
			filterField:   "email",
			filterValue:   []interface{}{"user1@example.com", "user2@example.com"},
			updateField:   "",
			updateValue:   nil,
			shouldError:   true,
		},
		{
			name: "Update with No Filters (All Records)",
			args: map[string]interface{}{
				"role": "user", // Just an update field
			},
			affectedCount: 3, // All records should be updated
			filterField:   "id",
			filterValue:   []interface{}{1, 2, 3},
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
				var params []interface{}

				// Build the filter part of the query
				if sliceVal, ok := tc.filterValue.([]interface{}); ok {
					// Handle array values (IN clause)
					placeholders := make([]string, len(sliceVal))
					for i := range placeholders {
						placeholders[i] = "?"
					}
					query = fmt.Sprintf("SELECT COUNT(*) FROM users WHERE `%s` IN (%s)",
						tc.filterField, strings.Join(placeholders, ","))

					for _, v := range sliceVal {
						params = append(params, v)
					}
				} else if mapVal, ok := tc.filterValue.(map[string]interface{}); ok {
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
						updateParams := append([]interface{}{}, params...)
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

// Test DELETE command
func TestDeleteCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]interface{}
		expectedIDs []int // IDs that should remain after deletion
		shouldError bool
	}{
		{
			name: "Delete Single User",
			args: map[string]interface{}{
				"id": 1,
			},
			expectedIDs: []int{2, 3},
			shouldError: false,
		},
		{
			name: "Delete Multiple Users",
			args: map[string]interface{}{
				"id": []interface{}{1, 2},
			},
			expectedIDs: []int{3},
			shouldError: false,
		},
		{
			name: "Delete Users in Range",
			args: map[string]interface{}{
				"id": map[string]interface{}{
					"range": []int{1, 2},
				},
			},
			expectedIDs: []int{3},
			shouldError: false,
		},
		{
			name: "Delete Non-existent User",
			args: map[string]interface{}{
				"id": 999,
			},
			expectedIDs: []int{1, 2, 3},
			shouldError: true,
		},
		{
			name: "Delete Without ID",
			args: map[string]interface{}{
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

// Test parser functions
func TestParserFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
		isError  bool
	}{
		{
			name:  "Parse Simple ID",
			input: "5",
			expected: map[string]interface{}{
				"id": 5,
			},
			isError: false,
		},
		{
			name:  "Parse Simple Object",
			input: "{name: 'John', age: 30}",
			expected: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			isError: false,
		},
		{
			name:  "Parse Array Values",
			input: "{id: 1}",
			expected: map[string]interface{}{
				"id": 1,
			},
			isError: false,
		},
		{
			name:  "Parse Range",
			input: "{id: (1, 10)}",
			expected: map[string]interface{}{
				"id": map[string]interface{}{
					"range": []int{1, 10},
				},
			},
			isError: false,
		},
		{
			name:  "Parse Multiple Field Assignment",
			input: "{[name, title] = 'Test'}",
			expected: map[string]interface{}{
				"name":  "Test",
				"title": "Test",
			},
			isError: false,
		},
		{
			name:     "Parse Invalid Input",
			input:    "invalid",
			expected: nil,
			isError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := pkg.ParseArg(tc.input)

			if tc.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// Test command parsing
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
	err := pkg.HandleCreate(testDB, map[string]interface{}{
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

				var argObj map[string]interface{}
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

// Test dynamic schema modification
func TestDynamicSchema(t *testing.T) {
	resetTable(t)

	// Try to create a user with fields that don't exist yet
	err := pkg.HandleCreate(testDB, map[string]interface{}{
		"name":     "Dynamic User",
		"email":    "dynamic@example.com",
		"age":      25,
		"location": "New York",
		"active":   true,
	}, true)
	assert.NoError(t, err)

	// Check if columns were created
	columns, err := getColumnsForTest(testDB)
	assert.NoError(t, err)

	// Convert to map for easier checking
	columnMap := make(map[string]bool)
	for _, col := range columns {
		columnMap[col] = true
	}

	// Verify required columns exist
	requiredColumns := []string{"id", "name", "email", "age", "location", "active"}
	for _, col := range requiredColumns {
		assert.True(t, columnMap[col], "Column %s should exist", col)
	}
}

// Helper function to get columns for tests
func getColumnsForTest(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW COLUMNS FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var field, fieldType, null, key, defaultVal, extra sql.NullString
		if err := rows.Scan(&field, &fieldType, &null, &key, &defaultVal, &extra); err != nil {
			return nil, err
		}
		columns = append(columns, field.String)
	}

	return columns, nil
}

// Test .env.test file setup
func TestCreateEnvTestFile(t *testing.T) {
	// Skip if file already exists
	if _, err := os.Stat(".env.test"); err == nil {
		t.Skip(".env.test file already exists")
	}

	// Create .env.test file with test credentials
	content := `TEST_DB_HOST=localhost
TEST_DB_USER=root
TEST_DB_PASS=1234
`
	err := os.WriteFile(".env.test", []byte(content), 0644)
	assert.NoError(t, err)
}

// TestOutputFormats tests both JSON and tabular output formats
func TestOutputFormats(t *testing.T) {
	resetTable(t)

	// Insert a test record
	err := pkg.HandleCreate(testDB, map[string]interface{}{
		"name":  "Format Test User",
		"email": "format@example.com",
	}, true) // JSON output
	assert.NoError(t, err)

	// Test JSON output (lowercase commands)
	err = pkg.HandleGet(testDB, nil, true)
	assert.NoError(t, err)

	// Test tabular output (uppercase commands)
	err = pkg.HandleGet(testDB, nil, false)
	assert.NoError(t, err)

	// Test update with JSON output
	err = pkg.HandleUpdate(testDB, map[string]interface{}{
		"id":   1,
		"name": "Updated Format User",
	}, true)
	assert.NoError(t, err)

	// Test update with tabular output
	err = pkg.HandleUpdate(testDB, map[string]interface{}{
		"id":    1,
		"email": "updated@example.com",
	}, false)
	assert.NoError(t, err)

	// Test delete with JSON output
	err = pkg.HandleDelete(testDB, map[string]interface{}{
		"id": 1,
	}, true)
	assert.NoError(t, err)
}
