package pkg

import (
	"database/sql"
	"fmt"
	"strings"
)

// HandleGet handles the GET command
func HandleGet(db *sql.DB, args map[string]any, useJsonOutput bool) error {
	if CurrentTable == "" {
		return fmt.Errorf("no table selected")
	}

	// --- Column selection support ---
	var selectColumns string = "*"
	var selectedCols []string
	if args != nil {
		if colsRaw, ok := args["_columns"]; ok {
			if cols, ok := colsRaw.([]string); ok && len(cols) > 0 {
				var quoted []string
				for _, c := range cols {
					quoted = append(quoted, fmt.Sprintf("`%s`", c))
					selectedCols = append(selectedCols, c)
				}
				selectColumns = strings.Join(quoted, ", ")
				delete(args, "_columns")
			} else if colsIface, ok := colsRaw.([]any); ok && len(colsIface) > 0 {
				var quoted []string
				for _, c := range colsIface {
					if s, ok := c.(string); ok {
						quoted = append(quoted, fmt.Sprintf("`%s`", s))
						selectedCols = append(selectedCols, s)
					}
				}
				if len(quoted) > 0 {
					selectColumns = strings.Join(quoted, ", ")
					delete(args, "_columns")
				}
			}
		}
	}
	if len(selectedCols) == 0 {
		// No explicit columns requested, use all columns
		allCols, err := getColumns(db)
		if err != nil {
			return err
		}
		selectedCols = allCols
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
		query = fmt.Sprintf("SELECT %s FROM %s", selectColumns, CurrentTable)
	} else {
		// Build WHERE clause
		var whereConditions []string

		for field, value := range args {
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
							values = append(values, val)
						default:
							// Convert other types to string
							values = append(values, fmt.Sprintf("%v", val))
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
			query = fmt.Sprintf("SELECT %s FROM %s WHERE %s",
				selectColumns, CurrentTable, strings.Join(whereConditions, " AND "))
		} else {
			// No conditions, get all
			query = fmt.Sprintf("SELECT %s FROM %s", selectColumns, CurrentTable)
		}
	}

	// Add LIKE condition if present
	if likeValue != nil {
		if len(selectedCols) == 0 {
			return fmt.Errorf("no columns found for LIKE clause")
		}
		var likeConditions []string
		likeStr := fmt.Sprintf("%v", likeValue)
		if !strings.Contains(likeStr, "%") {
			likeStr = "%" + likeStr + "%"
		}
		for _, col := range selectedCols {
			likeConditions = append(likeConditions, fmt.Sprintf("`%s` LIKE ?", col))
			values = append(values, likeStr)
		}
		likeClause := fmt.Sprintf("(%s)", strings.Join(likeConditions, " OR "))
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

	// DEBUG: Print the final query and values
	// fmt.Printf("[DEBUG] Executing query: %s\n", query)
	// fmt.Printf("[DEBUG] With values: %#v\n", values)

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
	// DEBUG: Print the columns returned
	// fmt.Printf("[DEBUG] Columns returned: %#v\n", columns)

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
