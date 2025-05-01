package pkg

import (
	"database/sql"
	"fmt"
	"strings"
)

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
