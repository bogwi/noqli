package pkg

import (
	"database/sql"
	"fmt"
	"strings"
)

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
				if len(sliceValue) == 0 {
					// Handle empty array
					whereConditions = append(whereConditions, "0=1") // No results should match
				} else {
					placeholders := make([]string, len(sliceValue))
					for i, v := range sliceValue {
						placeholders[i] = "?"
						// Convert numbers or other types to appropriate string representation if needed
						switch val := v.(type) {
						case int, int32, int64, float32, float64:
							// Keep numeric values as they are
							whereValues = append(whereValues, val)
						default:
							// Convert other types to string
							whereValues = append(whereValues, fmt.Sprintf("%v", val))
						}
					}
					whereConditions = append(whereConditions,
						fmt.Sprintf("`%s` IN (%s)", field, strings.Join(placeholders, ",")))
				}
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
