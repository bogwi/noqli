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

// HandleCreate handles the CREATE command
func HandleCreate(db *sql.DB, args map[string]any, useJsonOutput bool) error {
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
	var values []any

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
		// Colorized JSON output
		fmt.Printf("Created: %s\n", ColorJSON(args))
	} else {
		// MySQL-style tabular output
		fmt.Println("Query OK, 1 row affected")
		fmt.Printf("Last insert ID: %d\n", id)
	}

	return nil
}

// HandleGet handles the GET command
func HandleGet(db *sql.DB, args map[string]any, useJsonOutput bool) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	// Build query based on args
	var query string
	var values []any
	var orderByClause string

	// Check for ordering parameters
	if args != nil {
		if upValue, ok := args["up"]; ok {
			// Order ascending
			if colName, ok := upValue.(string); ok {
				orderByClause = fmt.Sprintf(" ORDER BY `%s` ASC", colName)
			}
			delete(args, "up")
		} else if upValue, ok := args["UP"]; ok {
			// Same for uppercase variant
			if colName, ok := upValue.(string); ok {
				orderByClause = fmt.Sprintf(" ORDER BY `%s` ASC", colName)
			}
			delete(args, "UP")
		}

		if downValue, ok := args["down"]; ok {
			// Order descending
			if colName, ok := downValue.(string); ok {
				orderByClause = fmt.Sprintf(" ORDER BY `%s` DESC", colName)
			}
			delete(args, "down")
		} else if downValue, ok := args["DOWN"]; ok {
			// Same for uppercase variant
			if colName, ok := downValue.(string); ok {
				orderByClause = fmt.Sprintf(" ORDER BY `%s` DESC", colName)
			}
			delete(args, "DOWN")
		}
	}

	// --- LIMIT/OFFSET support ---
	var limitClause string
	var limValue any
	var offValue any
	if args != nil {
		if v, ok := args["LIM"]; ok {
			limValue = v
			delete(args, "LIM")
		} else if v, ok := args["lim"]; ok {
			limValue = v
			delete(args, "lim")
		}
		if v, ok := args["OFF"]; ok {
			offValue = v
			delete(args, "OFF")
		} else if v, ok := args["off"]; ok {
			offValue = v
			delete(args, "off")
		}
		// Validate limit and offset are non-negative integers
		if limValue != nil {
			if limInt, ok := toInt(limValue); ok {
				if limInt < 0 {
					return fmt.Errorf("LIMIT must be non-negative")
				}
			} else {
				return fmt.Errorf("LIMIT must be an integer")
			}
		}
		if offValue != nil {
			if offInt, ok := toInt(offValue); ok {
				if offInt < 0 {
					return fmt.Errorf("OFFSET must be non-negative")
				}
			} else {
				return fmt.Errorf("OFFSET must be an integer")
			}
		}
		if limValue != nil {
			limitClause = " LIMIT ?"
			if offValue != nil {
				limitClause += " OFFSET ?"
			}
		}
	}

	// --- LIKE support ---
	var likeValue any
	if args != nil {
		if v, ok := args["LIKE"]; ok {
			likeValue = v
			delete(args, "LIKE")
		} else if v, ok := args["like"]; ok {
			likeValue = v
			delete(args, "like")
		}
	}

	if len(args) == 0 {
		// Get all records
		query = fmt.Sprintf("SELECT * FROM %s", CurrentTable)
	} else {
		// Build WHERE clause
		var whereConditions []string

		for field, value := range args {
			if sliceValue, ok := value.([]any); ok {
				// Handle array of values (IN clause)
				placeholders := make([]string, len(sliceValue))
				for i, v := range sliceValue {
					placeholders[i] = "?"
					values = append(values, v)
				}
				whereConditions = append(whereConditions,
					fmt.Sprintf("`%s` IN (%s)", field, strings.Join(placeholders, ",")))
			} else if mapValue, ok := value.(map[string]any); ok {
				// Handle range
				if rangeSlice, ok := mapValue["range"].([]int); ok && len(rangeSlice) == 2 {
					whereConditions = append(whereConditions,
						fmt.Sprintf("`%s` >= ? AND `%s` <= ?", field, field))
					values = append(values, rangeSlice[0], rangeSlice[1])
				} else {
					return fmt.Errorf("invalid range format for field %s", field)
				}
			} else {
				// Single value
				whereConditions = append(whereConditions, fmt.Sprintf("`%s` = ?", field))
				values = append(values, value)
			}
		}

		// Build the WHERE clause
		if len(whereConditions) > 0 {
			query = fmt.Sprintf("SELECT * FROM %s WHERE %s",
				CurrentTable, strings.Join(whereConditions, " AND "))
		} else {
			// No conditions, get all
			query = fmt.Sprintf("SELECT * FROM %s", CurrentTable)
		}
	}

	// Add LIKE condition if present
	if likeValue != nil {
		// Find all columns if not specified
		columns, err := getColumns(db)
		if err != nil {
			return err
		}

		if len(columns) == 0 {
			return fmt.Errorf("no columns found in table")
		}

		// Build LIKE conditions for all text-like columns
		var likeConditions []string

		// Convert likeValue to string and add wildcards if not already present
		likeStr := fmt.Sprintf("%v", likeValue)
		if !strings.Contains(likeStr, "%") {
			likeStr = "%" + likeStr + "%"
		}

		for _, col := range columns {
			likeConditions = append(likeConditions, fmt.Sprintf("`%s` LIKE ?", col))
			values = append(values, likeStr)
		}

		likeClause := fmt.Sprintf("(%s)", strings.Join(likeConditions, " OR "))

		// Append to existing query
		if strings.Contains(query, "WHERE") {
			query = fmt.Sprintf("%s AND %s", query, likeClause)
		} else {
			query = fmt.Sprintf("%s WHERE %s", query, likeClause)
		}
	}

	// Add ORDER BY clause if present
	if orderByClause != "" {
		query += orderByClause
	}
	// Add LIMIT/OFFSET clause if present
	if limitClause != "" {
		query += limitClause
	}

	// Execute query
	if limValue != nil && offValue != nil {
		values = append(values, limValue, offValue)
	} else if limValue != nil {
		values = append(values, limValue)
	}
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
	var results []map[string]any

	for rows.Next() {
		// Create a slice of any to hold the values
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		// Create a map for this row
		entry := make(map[string]any)
		for i, col := range columns {
			var v any
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
		// Colorized JSON output
		// Special case for single ID lookup for backward compatibility
		if id, ok := args["id"]; ok && len(args) == 1 && !isArrayOrRange(id) && len(results) == 1 {
			// Single result by ID
			fmt.Printf("Record: %s\n", ColorJSON(results[0]))
		} else {
			// Multiple results or non-ID query
			fmt.Printf("Records: %s\n", ColorJSON(results))
		}
	} else {
		// MySQL-style tabular output
		PrintTabularResults(columns, results)
	}

	return nil
}

// HandleUpdate handles the UPDATE command
func HandleUpdate(db *sql.DB, args map[string]any, useJsonOutput bool) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	if len(args) == 0 {
		return fmt.Errorf("UPDATE requires fields to update and filter conditions")
	}

	// Get existing columns to differentiate between filter and update columns
	existingCols, err := getColumns(db)
	if err != nil {
		return err
	}

	// Create maps for filter fields and update fields
	filterFields := make(map[string]any)
	updateFields := make(map[string]any)

	// First check: if there's only one field and it's an existing column with value as array/range, it's a filter
	if len(args) == 1 {
		for k, v := range args {
			if isArrayOrRange(v) {
				for _, col := range existingCols {
					if k == col {
						return fmt.Errorf("UPDATE requires fields to update (filter only provided)")
					}
				}
			}
		}
	}

	// Determine which fields are for filtering and which are for updating
	for k, v := range args {
		// Special handling for id field - always a filter
		if k == "id" {
			filterFields[k] = v
			continue
		}

		fieldExists := false
		for _, col := range existingCols {
			if k == col {
				fieldExists = true
				break
			}
		}

		// If field exists and value is array/range, it's a filter
		// Otherwise it's an update field (this includes new fields)
		if fieldExists && isArrayOrRange(v) {
			filterFields[k] = v
		} else {
			updateFields[k] = v
		}
	}

	// If no update fields, return error
	if len(updateFields) == 0 {
		return fmt.Errorf("UPDATE requires fields to update")
	}

	// If no filter fields, use all records (with warning)
	if len(filterFields) == 0 {
		fmt.Println("Warning: No filter conditions specified. This will update ALL records in the table.")
		fmt.Println("Do you want to continue? (y/N)")
		response := ScanForConfirmation()
		if strings.ToLower(response) != "y" {
			return fmt.Errorf("operation cancelled")
		}
	}

	// Ensure columns exist for update fields
	if err := ensureColumns(db, updateFields); err != nil {
		return err
	}

	// Build SET clause
	var setStatements []string
	var setValues []any

	for k, v := range updateFields {
		setStatements = append(setStatements, fmt.Sprintf("`%s` = ?", k))
		setValues = append(setValues, v)
	}

	// Build WHERE clause based on filter fields
	var whereClause string
	var whereValues []any

	if len(filterFields) > 0 {
		var whereConditions []string

		for field, value := range filterFields {
			if sliceValue, ok := value.([]any); ok {
				// Handle array of values (IN clause)
				placeholders := make([]string, len(sliceValue))
				for i, v := range sliceValue {
					placeholders[i] = "?"
					whereValues = append(whereValues, v)
				}
				whereConditions = append(whereConditions,
					fmt.Sprintf("`%s` IN (%s)", field, strings.Join(placeholders, ",")))
			} else if mapValue, ok := value.(map[string]any); ok {
				// Handle range
				if rangeSlice, ok := mapValue["range"].([]int); ok && len(rangeSlice) == 2 {
					whereConditions = append(whereConditions,
						fmt.Sprintf("`%s` >= ? AND `%s` <= ?", field, field))
					whereValues = append(whereValues, rangeSlice[0], rangeSlice[1])
				} else {
					return fmt.Errorf("invalid range format for field %s", field)
				}
			} else {
				// Single value
				whereConditions = append(whereConditions, fmt.Sprintf("`%s` = ?", field))
				whereValues = append(whereValues, value)
			}
		}

		whereClause = strings.Join(whereConditions, " AND ")
	}

	// Build query
	var query string
	var allValues []any

	// Add SET values
	allValues = append(allValues, setValues...)

	if whereClause != "" {
		query = fmt.Sprintf("UPDATE %s SET %s WHERE %s",
			CurrentTable,
			strings.Join(setStatements, ", "),
			whereClause)

		// Add WHERE values
		allValues = append(allValues, whereValues...)
	} else {
		query = fmt.Sprintf("UPDATE %s SET %s",
			CurrentTable,
			strings.Join(setStatements, ", "))
	}

	// Execute query
	result, err := db.Exec(query, allValues...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("no records matched the filter criteria")
	}

	if useJsonOutput {
		// Select the updated records for JSON output
		var selectQuery string
		if whereClause != "" {
			// The issue is here - when we update fields that are also used in the filter,
			// running the same query again won't find any matches

			// Original code - using the same whereClause as filter
			// selectQuery = fmt.Sprintf("SELECT * FROM %s WHERE %s", CurrentTable, whereClause)
			// return handleQueryAndDisplayResults(db, selectQuery, whereValues, len(filterFields) > 0, true)

			// Modified code - to fix the issue, we need to select rows by their IDs
			// First get the IDs of the affected rows
			var idQuery string
			if whereClause != "" {
				idQuery = fmt.Sprintf("SELECT id FROM %s WHERE %s", CurrentTable, whereClause)
			} else {
				idQuery = fmt.Sprintf("SELECT id FROM %s", CurrentTable)
			}

			rows, err := db.Query(idQuery, whereValues...)
			if err != nil {
				return err
			}
			defer rows.Close()

			var ids []any
			for rows.Next() {
				var id any
				if err := rows.Scan(&id); err != nil {
					return err
				}
				ids = append(ids, id)
			}

			// If we found matching rows, display them
			if len(ids) > 0 {
				placeholders := make([]string, len(ids))
				for i := range placeholders {
					placeholders[i] = "?"
				}
				selectQuery = fmt.Sprintf("SELECT * FROM %s WHERE id IN (%s)",
					CurrentTable, strings.Join(placeholders, ","))

				// Use these IDs to display the updated records
				return handleQueryAndDisplayResults(db, selectQuery, ids, true, true)
			} else {
				return fmt.Errorf("no records matched the filter criteria")
			}
		} else {
			selectQuery = fmt.Sprintf("SELECT * FROM %s LIMIT 10", CurrentTable)
			fmt.Printf("Updated %d record(s). Showing first 10:\n", affected)
			return handleQueryAndDisplayResults(db, selectQuery, nil, true, true)
		}
	} else {
		// MySQL-style tabular output
		fmt.Printf("Query OK, %d rows affected\n", affected)
		return nil
	}
}

// HandleDelete handles the DELETE command
func HandleDelete(db *sql.DB, args map[string]any, useJsonOutput bool) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	if args == nil || args["id"] == nil {
		return fmt.Errorf("DELETE requires an id field")
	}

	id := args["id"]

	var whereClause string
	var values []any

	// Handle different ID types
	if idSlice, ok := id.([]any); ok {
		// Multiple IDs
		placeholders := make([]string, len(idSlice))
		for i, v := range idSlice {
			placeholders[i] = "?"
			values = append(values, v)
		}
		whereClause = fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ","))
	} else if idMap, ok := id.(map[string]any); ok {
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
