package test

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/bogwi/noqli/pkg"
	"github.com/stretchr/testify/assert"
)

func TestGetCommandAggregate(t *testing.T) {
	resetTable(t)

	_, err := testDB.Exec(`
		INSERT INTO users (name, numeric_value, score, status) VALUES 
		('User 1', 10, 1.5, 'active'),
		('User 2', 20, 2.5, 'inactive'),
		('User 3', 30, 3.5, 'active'),
		('User 4', 40, 4.5, 'inactive'),
		('User 5', 10, 1.5, 'active'),
		('User 6', NULL, NULL, 'active'),
		('User 7', 20, 2.5, 'inactive')
	`)
	assert.NoError(t, err, "Failed to insert test data for aggregate test")

	testCases := []struct {
		name     string
		command  string // as user would type at CLI
		sql      string // direct SQL for validation
		params   []any
		col      string // column in output to check
		isFloat  bool
		jsonMode bool // true: lowercase command, false: UPPERCASE
	}{
		// MIN
		{"min numeric_value (json)", "get {MIN: 'numeric_value'}", "SELECT MIN(numeric_value) FROM users", nil, "min", false, true},
		{"min numeric_value (tabular)", "GET {MIN: 'numeric_value'}", "SELECT MIN(numeric_value) FROM users", nil, "min", false, false},
		// MAX
		{"max score (json)", "get {MAX: 'score'}", "SELECT MAX(score) FROM users", nil, "max", true, true},
		{"max score (tabular)", "GET {MAX: 'score'}", "SELECT MAX(score) FROM users", nil, "max", true, false},
		// AVG
		{"avg numeric_value (json)", "get {AVG: 'numeric_value'}", "SELECT AVG(numeric_value) FROM users", nil, "avg", true, true},
		{"avg numeric_value (tabular)", "GET {AVG: 'numeric_value'}", "SELECT AVG(numeric_value) FROM users", nil, "avg", true, false},
		// SUM
		{"sum numeric_value (json)", "get {SUM: 'numeric_value'}", "SELECT SUM(numeric_value) FROM users", nil, "sum", false, true},
		{"sum numeric_value (tabular)", "GET {SUM: 'numeric_value'}", "SELECT SUM(numeric_value) FROM users", nil, "sum", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get expected value from SQL
			var (
				intResult   sql.NullInt64
				floatResult sql.NullFloat64
			)
			row := testDB.QueryRow(tc.sql, tc.params...)
			if tc.isFloat {
				_ = row.Scan(&floatResult)
			} else {
				_ = row.Scan(&intResult)
			}

			// Capture stdout using os.Pipe
			r, w, _ := os.Pipe()
			oldStdout := os.Stdout
			os.Stdout = w

			// Parse command string as user would type
			cmdStr := tc.command
			args, err := pkg.ParseArg(strings.TrimPrefix(strings.TrimSpace(cmdStr), "get "))
			if err != nil {
				args, err = pkg.ParseArg(strings.TrimPrefix(strings.TrimSpace(cmdStr), "GET "))
			}
			assert.NoError(t, err, "ParseArg failed for: %s", cmdStr)

			// Set output mode
			useJson := tc.jsonMode
			err = pkg.HandleGet(testDB, args, useJson)
			assert.NoError(t, err, "HandleGet failed for: %s", cmdStr)
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			r.Close()

			output := buf.String()
			// Check output contains the expected value
			var expected string
			if tc.isFloat {
				expected = regexp.MustCompile(`\.0+$`).ReplaceAllString(fmt.Sprintf("%v", floatResult.Float64), "")
			} else {
				expected = fmt.Sprintf("%v", intResult.Int64)
			}
			assert.Contains(t, output, expected, "Output for %s should contain %s (got: %s)", tc.name, expected, output)
		})
	}
}
