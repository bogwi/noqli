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

	// Improved array parsing
	// Find all KEY: [ARRAY] patterns
	arrayRegex := regexp.MustCompile(`(\w+)\s*:\s*\[(.*?)\]`)
	arrayMatches = arrayRegex.FindAllStringSubmatch(trimmed, -1)

	for _, match := range arrayMatches {
		if len(match) >= 3 {
			key := match[1]
			arrayContent := match[2]

			// Remove the array pattern from the string
			fullMatch := match[0]
			trimmed = strings.Replace(trimmed, fullMatch, "", 1)

			// Split the array content by commas (respecting quotes)
			var arrayElements []any
			elements := splitRespectingQuotes(arrayContent, ',')

			for _, elem := range elements {
				elemTrimmed := strings.TrimSpace(elem)

				// Handle quoted strings
				if (strings.HasPrefix(elemTrimmed, "\"") && strings.HasSuffix(elemTrimmed, "\"")) ||
					(strings.HasPrefix(elemTrimmed, "'") && strings.HasSuffix(elemTrimmed, "'")) {
					// Remove quotes
					value := strings.Trim(elemTrimmed, `'"`)
					arrayElements = append(arrayElements, value)
				} else if num, err := strconv.Atoi(elemTrimmed); err == nil {
					// It's a number
					arrayElements = append(arrayElements, num)
				} else {
					// It's an unquoted string or identifier
					arrayElements = append(arrayElements, elemTrimmed)
				}
			}

			// Add the array to the result map
			result[key] = arrayElements
		}
	}

	// Process remaining key-value pairs
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

				// Skip array values we already processed
				if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") {
					continue
				}

				// Handle simple values
				if num, err := strconv.Atoi(valueStr); err == nil {
					result[key] = num
				} else {
					// If not a number, use as string
					result[key] = strings.Trim(valueStr, `'"`)
				}
			}
		} else {
			// If JSON parsing succeeds, merge the results
			for k, v := range jsonObj {
				// Skip array values we already processed
				if _, exists := result[k]; !exists {
					result[k] = v
				}
			}
		}
	}

	return result, nil
}

// Helper function to split a string by a delimiter respecting quotes
func splitRespectingQuotes(str string, delimiter rune) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, char := range str {
		switch {
		case char == '"' || char == '\'':
			if inQuotes && char == quoteChar {
				// Closing quote
				inQuotes = false
				quoteChar = rune(0)
			} else if !inQuotes {
				// Opening quote
				inQuotes = true
				quoteChar = char
			}
			current.WriteRune(char)
		case char == delimiter && !inQuotes:
			// Found delimiter outside quotes
			result = append(result, current.String())
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}

	// Add the last part
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
