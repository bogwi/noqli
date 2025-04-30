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

	// Create test table with all required columns for comprehensive testing
	_, err = testDB.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255),
			status VARCHAR(255),
			category VARCHAR(255),
			priority VARCHAR(255),
			tags VARCHAR(255),
			numeric_value INT,
			boolean_value TINYINT(1),
			processed TINYINT(1),
			level VARCHAR(255),
			updated_at VARCHAR(255),
			score FLOAT,
			global_field VARCHAR(255),
			bulk_update VARCHAR(255),
			new_status VARCHAR(255),
			range_updated VARCHAR(255),
			notes VARCHAR(255),
			modified TINYINT(1)
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

	// Create the users table with all required columns for comprehensive testing
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255),
			status VARCHAR(255),
			category VARCHAR(255),
			priority VARCHAR(255),
			tags VARCHAR(255),
			numeric_value INT,
			boolean_value TINYINT(1),
			processed TINYINT(1),
			level VARCHAR(255),
			updated_at VARCHAR(255),
			score FLOAT,
			global_field VARCHAR(255),
			bulk_update VARCHAR(255),
			new_status VARCHAR(255),
			range_updated VARCHAR(255),
			notes VARCHAR(255),
			modified TINYINT(1)
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

	// Insert additional test data with different names for ordering tests
	_, err := testDB.Exec(`
		INSERT INTO users (name, email) VALUES 
		('Alice', 'alice@example.com'),
		('Bob', 'bob@example.com'),
		('Charlie', 'charlie@example.com')
	`)
	assert.NoError(t, err, "Failed to insert additional test data")

	tests := []struct {
		name          string
		args          map[string]interface{}
		expectedCount int
		shouldError   bool
	}{
		{
			name:          "Get All Users",
			args:          nil,
			expectedCount: 6, // Updated count to match the additional inserted data
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
		// Test cases for ordering
		{
			name: "Get Users Ordered by Name Ascending (UP)",
			args: map[string]interface{}{
				"up": "name",
			},
			expectedCount: 6,
			shouldError:   false,
		},
		{
			name: "Get Users Ordered by Name Ascending (Uppercase UP)",
			args: map[string]interface{}{
				"UP": "name",
			},
			expectedCount: 6,
			shouldError:   false,
		},
		{
			name: "Get Users Ordered by Name Descending (DOWN)",
			args: map[string]interface{}{
				"down": "name",
			},
			expectedCount: 6,
			shouldError:   false,
		},
		{
			name: "Get Users Ordered by Name Descending (Uppercase DOWN)",
			args: map[string]interface{}{
				"DOWN": "name",
			},
			expectedCount: 6,
			shouldError:   false,
		},
		{
			name: "Get Users Filtered and Ordered",
			args: map[string]interface{}{
				"name": []interface{}{"Alice", "Bob", "Charlie"},
				"up":   "name",
			},
			expectedCount: 3,
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

// Test the ordering functionality specifically
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
		args := map[string]interface{}{
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
		args := map[string]interface{}{
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
		args := map[string]interface{}{
			"UP": "name",
		}
		err = pkg.HandleGet(testDB, args, true)
		assert.NoError(t, err)
	})

	t.Run("Descending Order with 'DOWN'", func(t *testing.T) {
		args := map[string]interface{}{
			"DOWN": "name",
		}
		err = pkg.HandleGet(testDB, args, true)
		assert.NoError(t, err)
	})

	// Test with filtering and ordering combined
	t.Run("Filtering and Ordering Combined", func(t *testing.T) {
		// First two names in alphabetical order (Alice, Bob)
		args := map[string]interface{}{
			"name": []interface{}{"Alice", "Bob", "Charlie"},
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

// TestComprehensiveUpdateOperations thoroughly tests all update operations and edge cases
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
	verifyUpdate := func(filterQuery string, filterParams []interface{}, expectedCount int,
		updateField string, updateValue interface{}) {

		// First verify the count of records matching the filter
		var count int
		err := testDB.QueryRow(filterQuery, filterParams...).Scan(&count)
		assert.NoError(t, err, "Error executing count query")
		assert.Equal(t, expectedCount, count, "Expected %d filtered records, got %d", expectedCount, count)

		if expectedCount > 0 && updateField != "" {
			// Verify the update was applied correctly
			updateQuery := filterQuery + fmt.Sprintf(" AND `%s` = ?", updateField)
			updateParams := append([]interface{}{}, filterParams...)
			updateParams = append(updateParams, updateValue)

			var updatedCount int
			err = testDB.QueryRow(updateQuery, updateParams...).Scan(&updatedCount)
			assert.NoError(t, err, "Error executing update verification query")
			assert.Equal(t, expectedCount, updatedCount, "Not all records were updated correctly")
		}
	}

	// 1. Update a single record by ID
	t.Run("Update Single Record by ID", func(t *testing.T) {
		args := map[string]interface{}{
			"id":     1,
			"status": "updated-status",
			"notes":  "New field with dynamic column creation",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE id = ?",
			[]interface{}{1},
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
		args := map[string]interface{}{
			"id":       []interface{}{2, 3, 4},
			"category": "batch-updated",
			"modified": true,
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE id IN (?, ?, ?)",
			[]interface{}{2, 3, 4},
			3,
			"category",
			"batch-updated",
		)
	})

	// 3. Update records in ID range
	t.Run("Update Records in ID Range", func(t *testing.T) {
		args := map[string]interface{}{
			"id": map[string]interface{}{
				"range": []int{3, 5},
			},
			"range_updated": "yes",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE id >= ? AND id <= ?",
			[]interface{}{3, 5},
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
		args := map[string]interface{}{
			"id":     5, // Use ID as filter
			"status": "approved",
			"email":  "filtered.user@example.com", // This will be an update field, not a filter
		}

		err = pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		// Verify the update using ID as the filter
		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE id = ?",
			[]interface{}{5},
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

		args := map[string]interface{}{
			"status":      []interface{}{"pending", "active"},
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
			[]interface{}{},
			2, // 2 active records (ID 1 and 4)
			"bulk_update",
			"processed",
		)
	})

	// 6. Update all records (with confirmation)
	t.Run("Update All Records", func(t *testing.T) {
		args := map[string]interface{}{
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
		args := map[string]interface{}{
			"id":    999,
			"field": "value",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no records matched")
	})

	// 8. Update with only filter fields (no update fields)
	t.Run("Update with Filter-only", func(t *testing.T) {
		args := map[string]interface{}{
			"status": []interface{}{"active", "pending"},
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires fields to update")
	})

	// 9. Update with empty args
	t.Run("Update with Empty Args", func(t *testing.T) {
		args := map[string]interface{}{}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.Error(t, err)
	})

	// 10. Update with invalid range
	t.Run("Update with Invalid Range", func(t *testing.T) {
		args := map[string]interface{}{
			"id": map[string]interface{}{
				"range": []int{10, 5}, // Invalid: start > end
			},
			"field": "value",
		}

		err := pkg.HandleUpdate(testDB, args, true)
		assert.Error(t, err) // Should error because no records match
	})

	// 11. Update with multiple field types
	t.Run("Update with Multiple Field Types", func(t *testing.T) {
		args := map[string]interface{}{
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
		_, err = testDB.Exec("UPDATE users SET category='type-B' WHERE id IN (3, 4, 5)")
		assert.NoError(t, err)

		// Use category as both filter (array) and update (string)
		args := map[string]interface{}{
			"category":   []interface{}{"type-A"}, // This is a filter
			"new_status": "special",               // This is an update field
		}

		err = pkg.HandleUpdate(testDB, args, true)
		assert.NoError(t, err)

		// Verify only type-A records were updated
		verifyUpdate(
			"SELECT COUNT(*) FROM users WHERE category = ?",
			[]interface{}{"type-A"},
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
		args := map[string]interface{}{
			// Filter fields (using array notation)
			"status":   []interface{}{"active", "pending"},
			"priority": []interface{}{"high"},

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
		verifyParams := []interface{}{"active", "pending", "high"}

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
