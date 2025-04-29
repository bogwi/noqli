package pkg

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Global variables to track current database and table
var CurrentDB string
var CurrentTable string

// GetCommandRegex returns the regex used to parse NoQLi commands
func GetCommandRegex() *regexp.Regexp {
	return regexp.MustCompile(`(?i)^(CREATE|GET|UPDATE|DELETE|USE)\s*(.*)$`)
}

// GetUseCommandRegex returns the regex for USE commands
func GetUseCommandRegex() *regexp.Regexp {
	return regexp.MustCompile(`(?i)^USE\s+(.+)$`)
}

// IsGetDbsCommand checks if the command is GET dbs
func IsGetDbsCommand(command string, args string) bool {
	return strings.ToUpper(command) == "GET" && strings.ToLower(strings.TrimSpace(args)) == "dbs"
}

// IsGetTablesCommand checks if the command is GET tables
func IsGetTablesCommand(command string, args string) bool {
	return strings.ToUpper(command) == "GET" && strings.ToLower(strings.TrimSpace(args)) == "tables"
}

// ParseArg parses the argument string into a map
func ParseArg(str string) (map[string]any, error) {
	if str == "" {
		return nil, nil
	}

	trimmed := strings.TrimSpace(str)

	// Handle simple numeric ID case (e.g., GET 14)
	if matches, _ := regexp.MatchString(`^\d+$`, trimmed); matches {
		id, _ := strconv.Atoi(trimmed)
		return map[string]any{"id": id}, nil
	}

	// Handle object notation
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return parseObjectNotation(trimmed)
	}

	return nil, fmt.Errorf("invalid argument format")
}

// DisplayPrompt shows the appropriate prompt based on current selections
func DisplayPrompt() string {
	prompt := "noqli"
	if CurrentDB != "" {
		prompt += ":" + CurrentDB
		if CurrentTable != "" {
			prompt += ":" + CurrentTable
		}
	}
	prompt += "> "
	return prompt
}

// parseObjectNotation handles the '{field1: value, field2: value}' syntax
func parseObjectNotation(str string) (map[string]any, error) {
	// Remove surrounding braces
	trimmed := strings.TrimSpace(str[1 : len(str)-1])

	// Result map
	result := make(map[string]any)

	// Handle array assignments like [field1,field2] = value
	arrayFieldRegex := regexp.MustCompile(`\[([^\]]+)\]\s*=\s*([^,}]+)`)
	arrayMatches := arrayFieldRegex.FindAllStringSubmatch(trimmed, -1)

	// Process array field assignments
	for _, match := range arrayMatches {
		fullMatch := match[0]
		fields := strings.Split(match[1], ",")
		valueStr := strings.TrimSpace(match[2])

		// Replace in the original string
		trimmed = strings.Replace(trimmed, fullMatch, "", 1)

		// Parse the value
		var value any

		// Try as JSON
		if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
			// If not JSON, use string with quotes removed
			value = strings.Trim(valueStr, `'"`)
		}

		// Assign to all fields
		for _, field := range fields {
			result[strings.TrimSpace(field)] = value
		}
	}

	// Process ID range syntax: id: (start, stop)
	rangeRegex := regexp.MustCompile(`id\s*:\s*\(([^,]+),([^)]+)\)`)
	if rangeMatches := rangeRegex.FindStringSubmatch(trimmed); len(rangeMatches) > 0 {
		fullMatch := rangeMatches[0]
		start, err := strconv.Atoi(strings.TrimSpace(rangeMatches[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid range start: %v", err)
		}

		end, err := strconv.Atoi(strings.TrimSpace(rangeMatches[2]))
		if err != nil {
			return nil, fmt.Errorf("invalid range end: %v", err)
		}

		result["id"] = map[string]any{
			"range": []int{start, end},
		}

		// Replace in the original string
		trimmed = strings.Replace(trimmed, fullMatch, "", 1)
	}

	// Clean up the remaining string
	trimmed = strings.TrimSpace(trimmed)
	trimmed = regexp.MustCompile(`,\s*,`).ReplaceAllString(trimmed, ",")
	trimmed = regexp.MustCompile(`^,|,$`).ReplaceAllString(trimmed, "")

	// If we still have content, parse it as a regular object
	if trimmed != "" {
		// Try to parse as JSON
		jsonStr := "{" + strings.Replace(trimmed, "'", "\"", -1) + "}"
		var jsonObj map[string]any

		if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err != nil {
			// If JSON parsing fails, try a more manual approach
			keyValuePairs := strings.Split(trimmed, ",")
			for _, pair := range keyValuePairs {
				parts := strings.SplitN(pair, ":", 2)
				if len(parts) != 2 {
					continue
				}

				key := strings.TrimSpace(parts[0])
				valueStr := strings.TrimSpace(parts[1])

				// Handle arrays in bracket notation
				if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") {
					arrayStr := valueStr[1 : len(valueStr)-1]
					elements := strings.Split(arrayStr, ",")

					// Try to convert elements to integers
					var intArray []any
					for _, elem := range elements {
						trimmedElem := strings.TrimSpace(elem)
						if num, err := strconv.Atoi(trimmedElem); err == nil {
							intArray = append(intArray, num)
						} else {
							// If not a number, use as string
							intArray = append(intArray, strings.Trim(trimmedElem, `'"`))
						}
					}

					result[key] = intArray
				} else {
					// Try to convert to number
					if num, err := strconv.Atoi(valueStr); err == nil {
						result[key] = num
					} else {
						// If not a number, use as string
						result[key] = strings.Trim(valueStr, `'"`)
					}
				}
			}
		} else {
			// If JSON parsing succeeds, merge the results
			for k, v := range jsonObj {
				result[k] = v
			}
		}
	}

	return result, nil
}
