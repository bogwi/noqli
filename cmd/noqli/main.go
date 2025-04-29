package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bogwi/noqli/pkg"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	// Connect to database
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		fmt.Println("Error connecting to database:", err)
		return
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		fmt.Println("Error pinging database:", err)
		return
	}
	fmt.Println("Connected to MySQL")

	// Set initial database from env
	pkg.CurrentDB = os.Getenv("DB_NAME")

	// Start CLI
	fmt.Println("NoQLi CLI. Type EXIT to quit.")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Display prompt based on current db/table selection
		fmt.Print(pkg.DisplayPrompt())

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		if strings.ToUpper(strings.TrimSpace(line)) == "EXIT" {
			break
		}

		if err := handleCommand(db, line); err != nil {
			fmt.Println("Error:", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input:", err)
	}
}

func handleCommand(db *sql.DB, line string) error {
	trimmed := strings.TrimSpace(line)

	// Check for USE command first
	useCommandRegex := pkg.GetUseCommandRegex()
	useMatches := useCommandRegex.FindStringSubmatch(trimmed)

	if useMatches != nil {
		// Handle USE command
		return handleUse(db, useMatches[1])
	}

	// Handle other commands
	re := pkg.GetCommandRegex()
	matches := re.FindStringSubmatch(trimmed)

	if matches == nil {
		return fmt.Errorf("invalid command. Use CREATE, GET, UPDATE, DELETE, USE, or EXIT")
	}

	originalCommand := matches[1]
	command := strings.ToUpper(originalCommand)
	args := matches[2]

	// Check if command was originally uppercase (for formatting choice)
	useJsonOutput := originalCommand != command

	// Special handling for GET dbs and GET tables
	if pkg.IsGetDbsCommand(command, args) {
		return handleGetDatabases(db, line)
	} else if pkg.IsGetTablesCommand(command, args) {
		return handleGetTables(db, line)
	}

	// Handle regular CRUD operations
	var argObj map[string]interface{}
	var err error

	if args != "" {
		argObj, err = pkg.ParseArg(args)
		if err != nil {
			return fmt.Errorf("could not parse argument object: %v", err)
		}
	}

	// Ensure a table is selected before executing CRUD operations
	if pkg.CurrentTable == "" && (command == "CREATE" || command == "GET" || command == "UPDATE" || command == "DELETE") {
		return fmt.Errorf("no table selected. Use 'USE table_name' to select a table")
	}

	switch command {
	case "CREATE":
		return pkg.HandleCreate(db, argObj, useJsonOutput)
	case "GET":
		return pkg.HandleGet(db, argObj, useJsonOutput)
	case "UPDATE":
		return pkg.HandleUpdate(db, argObj, useJsonOutput)
	case "DELETE":
		return pkg.HandleDelete(db, argObj, useJsonOutput)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleUse handles the USE command to select database or table
func handleUse(db *sql.DB, name string) error {
	// Check if name is a database
	var exists int
	err := db.QueryRow("SELECT 1 FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", name).Scan(&exists)
	if err == nil {
		// It's a database, switch to it
		_, err = db.Exec("USE " + name)
		if err != nil {
			return fmt.Errorf("failed to switch to database %s: %v", name, err)
		}
		pkg.CurrentDB = name
		pkg.CurrentTable = "" // Reset table selection when changing database
		fmt.Printf("Switched to database '%s'\n", name)
		return nil
	}

	// Not a database, check if it's a table in the current database
	if pkg.CurrentDB == "" {
		return fmt.Errorf("no database selected. Use 'USE database_name' first")
	}

	err = db.QueryRow(fmt.Sprintf("SELECT 1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"),
		pkg.CurrentDB, name).Scan(&exists)
	if err == nil {
		// It's a table, select it
		pkg.CurrentTable = name
		fmt.Printf("Using table '%s'\n", name)
		return nil
	} else if err == sql.ErrNoRows {
		return fmt.Errorf("table '%s' does not exist in database '%s'", name, pkg.CurrentDB)
	} else {
		return err
	}
}

// handleGetDatabases shows all available databases
func handleGetDatabases(db *sql.DB, line string) error {
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return err
	}
	defer rows.Close()

	// Check if the command was in uppercase (for formatting choice)
	useJsonOutput := false
	for _, r := range line {
		if r == 'g' || r == 'G' {
			useJsonOutput = (r == 'g')
			break
		}
	}

	if useJsonOutput {
		// JSON output
		var databases []string
		for rows.Next() {
			var dbName string
			if err := rows.Scan(&dbName); err != nil {
				return err
			}
			databases = append(databases, dbName)
		}

		resultJSON, _ := json.MarshalIndent(databases, "", "  ")
		fmt.Printf("Databases: %s\n", resultJSON)
	} else {
		// MySQL-style tabular output
		var databases []map[string]interface{}
		for rows.Next() {
			var dbName string
			if err := rows.Scan(&dbName); err != nil {
				return err
			}
			databases = append(databases, map[string]interface{}{"Database": dbName})
		}

		columns := []string{"Database"}
		pkg.PrintTabularResults(columns, databases)
	}

	return nil
}

// handleGetTables shows all tables in the current database
func handleGetTables(db *sql.DB, line string) error {
	if pkg.CurrentDB == "" {
		return fmt.Errorf("no database selected. Use 'USE database_name' first")
	}

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return err
	}
	defer rows.Close()

	// Check if the command was in uppercase (for formatting choice)
	useJsonOutput := false
	for _, r := range line {
		if r == 'g' || r == 'G' {
			useJsonOutput = (r == 'g')
			break
		}
	}

	if useJsonOutput {
		// JSON output
		var tables []string
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				return err
			}
			tables = append(tables, tableName)
		}

		resultJSON, _ := json.MarshalIndent(tables, "", "  ")
		fmt.Printf("Tables in %s: %s\n", pkg.CurrentDB, resultJSON)
	} else {
		// MySQL-style tabular output
		var tables []map[string]interface{}
		tableTitleColumn := fmt.Sprintf("Tables_in_%s", pkg.CurrentDB)

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				return err
			}
			tables = append(tables, map[string]interface{}{tableTitleColumn: tableName})
		}

		columns := []string{tableTitleColumn}
		pkg.PrintTabularResults(columns, tables)
	}

	return nil
}
