package pkg

import (
	"database/sql"
	"encoding/json"
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
func ensureColumns(db *sql.DB, fields map[string]interface{}) error {
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

// HandleCreate handles the CREATE command
func HandleCreate(db *sql.DB, args map[string]interface{}, useJsonOutput bool) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	if len(args) == 0 {
		return fmt.Errorf("CREATE requires fields to insert")
	}

	// Ensure columns exist
	if err := ensureColumns(db, args); err != nil {
		return err
	}

	// Build query
	var fields []string
	var placeholders []string
	var values []interface{}

	for k, v := range args {
		fields = append(fields, fmt.Sprintf("`%s`", k))
		placeholders = append(placeholders, "?")
		values = append(values, v)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		CurrentTable,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	// Execute query
	result, err := db.Exec(query, values...)
	if err != nil {
		return err
	}

	// Get inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Output result
	args["id"] = id

	if useJsonOutput {
		// JSON output (original)
		resultJSON, _ := json.MarshalIndent(args, "", "  ")
		fmt.Printf("Created: %s\n", resultJSON)
	} else {
		// MySQL-style tabular output
		fmt.Println("Query OK, 1 row affected")
		fmt.Printf("Last insert ID: %d\n", id)
	}

	return nil
}

// HandleGet handles the GET command
func HandleGet(db *sql.DB, args map[string]interface{}, useJsonOutput bool) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	var query string
	var values []interface{}

	if args == nil {
		// Get all records
		query = fmt.Sprintf("SELECT * FROM %s", CurrentTable)
	} else if id, ok := args["id"]; ok {
		// Check if id is an array
		if idSlice, ok := id.([]interface{}); ok {
			// Multiple IDs
			placeholders := make([]string, len(idSlice))
			for i, v := range idSlice {
				placeholders[i] = "?"
				values = append(values, v)
			}
			query = fmt.Sprintf("SELECT * FROM %s WHERE id IN (%s)", CurrentTable, strings.Join(placeholders, ","))
		} else if idMap, ok := id.(map[string]interface{}); ok {
			// Range query
			if rangeSlice, ok := idMap["range"].([]int); ok && len(rangeSlice) == 2 {
				query = fmt.Sprintf("SELECT * FROM %s WHERE id >= ? AND id <= ?", CurrentTable)
				values = append(values, rangeSlice[0], rangeSlice[1])
			} else {
				return fmt.Errorf("invalid range format")
			}
		} else {
			// Single ID
			query = fmt.Sprintf("SELECT * FROM %s WHERE id = ?", CurrentTable)
			values = append(values, id)
		}
	} else {
		return fmt.Errorf("invalid GET arguments")
	}

	// Execute query
	rows, err := db.Query(query, values...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// Prepare results
	var results []map[string]interface{}

	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		// Create a map for this row
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]

			// Convert to appropriate Go type
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

	// Output results
	if len(results) == 0 {
		fmt.Println("No records found")
		return nil
	}

	if useJsonOutput {
		// JSON output (original)
		if _, ok := args["id"]; ok && len(results) == 1 && !isArrayOrRange(args["id"]) {
			// Single result
			resultJSON, _ := json.MarshalIndent(results[0], "", "  ")
			fmt.Printf("Record: %s\n", resultJSON)
		} else {
			// Multiple results
			resultJSON, _ := json.MarshalIndent(results, "", "  ")
			fmt.Printf("Records: %s\n", resultJSON)
		}
	} else {
		// MySQL-style tabular output
		PrintTabularResults(columns, results)
	}

	return nil
}

// HandleUpdate handles the UPDATE command
func HandleUpdate(db *sql.DB, args map[string]interface{}, useJsonOutput bool) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	if args == nil || args["id"] == nil {
		return fmt.Errorf("UPDATE requires an id field")
	}

	id := args["id"]
	delete(args, "id") // Remove id from fields to update

	if len(args) == 0 {
		return fmt.Errorf("UPDATE requires fields to update")
	}

	// Ensure columns exist
	if err := ensureColumns(db, args); err != nil {
		return err
	}

	// Build query
	var setStatements []string
	var values []interface{}

	for k, v := range args {
		setStatements = append(setStatements, fmt.Sprintf("`%s` = ?", k))
		values = append(values, v)
	}

	var whereClause string
	var idValues []interface{}

	// Handle different ID types
	if idSlice, ok := id.([]interface{}); ok {
		// Multiple IDs
		placeholders := make([]string, len(idSlice))
		for i, v := range idSlice {
			placeholders[i] = "?"
			idValues = append(idValues, v)
		}
		whereClause = fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ","))
	} else if idMap, ok := id.(map[string]interface{}); ok {
		// Range query
		if rangeSlice, ok := idMap["range"].([]int); ok && len(rangeSlice) == 2 {
			whereClause = "id >= ? AND id <= ?"
			idValues = append(idValues, rangeSlice[0], rangeSlice[1])
		} else {
			return fmt.Errorf("invalid range format")
		}
	} else {
		// Single ID
		whereClause = "id = ?"
		idValues = append(idValues, id)
	}

	// Combine values
	values = append(values, idValues...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		CurrentTable,
		strings.Join(setStatements, ", "),
		whereClause,
	)

	// Execute query
	result, err := db.Exec(query, values...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("record(s) not found")
	}

	if useJsonOutput {
		// Select the updated records for JSON output
		selectQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s", CurrentTable, whereClause)
		return handleQueryAndDisplayResults(db, selectQuery, idValues, isArrayOrRange(id), true)
	} else {
		// MySQL-style tabular output
		fmt.Printf("Query OK, %d rows affected\n", affected)
		return nil
	}
}

// HandleDelete handles the DELETE command
func HandleDelete(db *sql.DB, args map[string]interface{}, useJsonOutput bool) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	if args == nil || args["id"] == nil {
		return fmt.Errorf("DELETE requires an id field")
	}

	id := args["id"]

	var whereClause string
	var values []interface{}

	// Handle different ID types
	if idSlice, ok := id.([]interface{}); ok {
		// Multiple IDs
		placeholders := make([]string, len(idSlice))
		for i, v := range idSlice {
			placeholders[i] = "?"
			values = append(values, v)
		}
		whereClause = fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ","))
	} else if idMap, ok := id.(map[string]interface{}); ok {
		// Range query
		if rangeSlice, ok := idMap["range"].([]int); ok && len(rangeSlice) == 2 {
			whereClause = "id >= ? AND id <= ?"
			values = append(values, rangeSlice[0], rangeSlice[1])
		} else {
			return fmt.Errorf("invalid range format")
		}
	} else {
		// Single ID
		whereClause = "id = ?"
		values = append(values, id)
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", CurrentTable, whereClause)

	// Execute query
	result, err := db.Exec(query, values...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("record(s) not found")
	}

	if useJsonOutput {
		// JSON output (original)
		fmt.Printf("Deleted %d record(s)\n", affected)
	} else {
		// MySQL-style tabular output
		fmt.Printf("Query OK, %d rows affected\n", affected)
	}

	return nil
}

// Helper function to determine if ID is an array or range
func isArrayOrRange(id interface{}) bool {
	_, isSlice := id.([]interface{})
	_, isMap := id.(map[string]interface{})
	return isSlice || isMap
}

// handleQueryAndDisplayResults executes a query and displays the results
func handleQueryAndDisplayResults(db *sql.DB, query string, values []interface{}, isMultiple bool, useJsonOutput bool) error {
	rows, err := db.Query(query, values...)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
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
		// JSON output (original)
		if !isMultiple && len(results) == 1 {
			resultJSON, _ := json.MarshalIndent(results[0], "", "  ")
			fmt.Println(string(resultJSON))
		} else {
			resultJSON, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(resultJSON))
		}
	} else {
		// MySQL-style tabular output
		PrintTabularResults(columns, results)
	}

	return nil
}

// printTabularResults prints results in a MySQL-like tabular format
func PrintTabularResults(columns []string, results []map[string]interface{}) {
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
