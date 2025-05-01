package pkg

import (
	"database/sql"
	"fmt"
	"strings"
)

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
