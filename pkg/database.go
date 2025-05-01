package pkg

import (
	"database/sql"
	"fmt"
	"strings"
)

// getColumns retrieves all column names from the current table
func getColumns(db *sql.DB) ([]string, error) {
	if CurrentTable == "" {
		return nil, fmt.Errorf("no table selected")
	}

	rows, err := db.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", CurrentTable))
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

// ensureColumns creates columns in the table if they don't exist
func ensureColumns(db *sql.DB, fields map[string]any) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	existingCols, err := getColumns(db)
	if err != nil {
		return err
	}

	// Create a map for faster lookup
	colMap := make(map[string]bool)
	for _, col := range existingCols {
		colMap[col] = true
	}

	// Check if each field exists, create if not
	for key := range fields {
		if key == "id" {
			continue // Skip id field
		}

		if !colMap[key] {
			_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN `%s` VARCHAR(255)", CurrentTable, key))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper function to determine if ID is an array or range
func isArrayOrRange(id any) bool {
	_, isSlice := id.([]any)
	_, isMap := id.(map[string]any)
	return isSlice || isMap
}

// handleQueryAndDisplayResults executes a query and displays the results
func handleQueryAndDisplayResults(db *sql.DB, query string, values []any, isMultiple bool, useJsonOutput bool) error {
	rows, err := db.Query(query, values...)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var results []map[string]any

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		entry := make(map[string]any)
		for i, col := range columns {
			var v any
			val := values[i]

			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}

			entry[col] = v
		}

		results = append(results, entry)
	}

	if len(results) == 0 {
		return fmt.Errorf("no records found")
	}

	if useJsonOutput {
		// Colorized JSON output
		if !isMultiple && len(results) == 1 {
			fmt.Println(ColorJSON(results[0]))
		} else {
			fmt.Println(ColorJSON(results))
		}
	} else {
		// MySQL-style tabular output
		PrintTabularResults(columns, results)
	}

	return nil
}

// printTabularResults prints results in a MySQL-like tabular format
func PrintTabularResults(columns []string, results []map[string]any) {
	if len(results) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make(map[string]int)
	for _, col := range columns {
		colWidths[col] = len(col)
	}

	// Find the max width for each column
	for _, row := range results {
		for col, val := range row {
			valStr := fmt.Sprintf("%v", val)
			if len(valStr) > colWidths[col] {
				colWidths[col] = len(valStr)
			}
		}
	}

	// Print header
	fmt.Println()
	for _, col := range columns {
		fmt.Printf("| %-*s ", colWidths[col], col)
	}
	fmt.Println("|")

	// Print separator
	for _, col := range columns {
		fmt.Print("+")
		for i := 0; i < colWidths[col]+2; i++ {
			fmt.Print("-")
		}
	}
	fmt.Println("+")

	// Print rows
	for _, row := range results {
		for _, col := range columns {
			val := row[col]
			fmt.Printf("| %-*v ", colWidths[col], val)
		}
		fmt.Println("|")
	}

	// Print row count
	fmt.Printf("\n%d rows in set\n", len(results))
}

// Default function for user input confirmation
var ScanForConfirmation = func() string {
	var response string
	fmt.Scanln(&response)
	return response
}

// Helper to convert any to int
func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}

// getTextColumns returns only the text columns for the current table
func getTextColumns(db *sql.DB) ([]string, error) {
	if CurrentTable == "" {
		return nil, fmt.Errorf("no table selected")
	}

	rows, err := db.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", CurrentTable))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var textColumns []string
	for rows.Next() {
		var field, fieldType, null, key, defaultVal, extra sql.NullString
		if err := rows.Scan(&field, &fieldType, &null, &key, &defaultVal, &extra); err != nil {
			return nil, err
		}
		// Check if the type is a text type
		t := strings.ToUpper(fieldType.String)
		if strings.Contains(t, "CHAR") || strings.Contains(t, "TEXT") || strings.Contains(t, "ENUM") || strings.Contains(t, "SET") {
			textColumns = append(textColumns, field.String)
		}
	}
	return textColumns, nil
}
