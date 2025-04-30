package test

import (
	"database/sql"
	"fmt"
	"os"
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
