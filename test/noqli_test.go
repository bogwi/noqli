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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := pkg.HandleGet(testDB, tc.args, true)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Note: Since HandleGet prints results rather than returning them,
			// we can't easily verify the actual results in this test.
			// In a real implementation, we might modify HandleGet to return results
			// or use a mock stdout to capture the output.
		})
	}
}

// Test UPDATE command
func TestUpdateCommand(t *testing.T) {
	resetTable(t)
	insertTestData(t)

	tests := []struct {
		name        string
		args        map[string]interface{}
		affectedIDs []int
		shouldError bool
	}{
		{
			name: "Update Single User",
			args: map[string]interface{}{
				"id":    1,
				"name":  "Updated Name",
				"email": "updated@example.com",
			},
			affectedIDs: []int{1},
			shouldError: false,
		},
		{
			name: "Update Multiple Users",
			args: map[string]interface{}{
				"id":     []interface{}{2, 3},
				"status": "inactive",
			},
			affectedIDs: []int{2, 3},
			shouldError: false,
		},
		{
			name: "Update Users in Range",
			args: map[string]interface{}{
				"id": map[string]interface{}{
					"range": []int{1, 3},
				},
				"updated": true,
			},
			affectedIDs: []int{1, 2, 3},
			shouldError: false,
		},
		{
			name: "Update Non-existent User",
			args: map[string]interface{}{
				"id":   999,
				"name": "Won't Update",
			},
			affectedIDs: []int{},
			shouldError: true,
		},
		{
			name: "Update Without ID",
			args: map[string]interface{}{
				"name": "Missing ID",
			},
			affectedIDs: []int{},
			shouldError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := pkg.HandleUpdate(testDB, tc.args, true)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify updates for each affected ID
				for _, id := range tc.affectedIDs {
					var count int
					query := "SELECT COUNT(*) FROM users WHERE id = ?"
					params := []interface{}{id}

					// Add field conditions if present
					for k, v := range tc.args {
						if k != "id" {
							query += fmt.Sprintf(" AND `%s` = ?", k)
							params = append(params, v)
						}
					}

					err := testDB.QueryRow(query, params...).Scan(&count)
					assert.NoError(t, err)
					assert.Equal(t, 1, count, "Update not applied for ID %d", id)
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
