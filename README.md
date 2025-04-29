# NoQLi (Go Version)

NoQLi (pronounced "no-klee") is an interactive MySQL command-line interface with a flexible query syntax that combines the simplicity of NoSQL-style commands with the power of a relational database.

## Features

- Interactive command-line interface for MySQL databases
- Simplified CRUD operations (CREATE, GET, UPDATE, DELETE)
- Support for complex queries including arrays and ranges
- Dynamic schema modification - automatically creates columns when needed
- Flexible query syntax with intuitive object notation
- Database and table selection commands with dynamic prompt
- Works with any MySQL database and table
- Dual output format: JSON or MySQL-style tabular format

## Installation

### Option 1: Clone and Build

1. Install Go (if not already installed):
   - Download from [golang.org/dl](https://golang.org/dl/)
   - Follow the installation instructions for your operating system

2. Clone the repository:
   ```
   git clone https://github.com/bogwi/noqli.git
   cd noqli
   ```

3. Install dependencies and build:
   ```
   go mod download
   make build
   ```

4. Create a `.env` file with your MySQL credentials:
   ```
   # Copy the example file
   cp env.example .env
   
   # Edit with your MySQL credentials
   DB_HOST=localhost
   DB_USER=your_username
   DB_PASSWORD=your_password
   DB_NAME=your_database
   ```

5. Run the application:
   ```
   ./bin/noqli
   ```

### Option 2: Go Install

You can install directly from GitHub:

```
go install github.com/bogwi/noqli/cmd/noqli@latest
```

Then create a `.env` file in the directory where you run the command, with your database credentials.

## Project Structure

```
noqli/
├── cmd/
│   └── noqli/        # Main application
│       └── main.go
├── pkg/              # Core functionality
│   ├── database.go   # Database operations
│   └── parser.go     # Command parsing
├── test/             # Test files
├── bin/              # Compiled binaries
├── .env              # Environment configuration
├── .env.test         # Test environment configuration
├── go.mod            # Go module definition
├── go.sum            # Go module checksums
├── Makefile          # Build automation
└── README.md         # This file
```

## Development

### Build

```
make build
```

### Test

```
make test
```

### Clean

```
make clean
```

## Usage

Start the CLI:

```
./bin/noqli
```

### Command Syntax

NoQLi supports several commands:

#### Output Formats

NoQLi now supports two output formats:
- **JSON format**: Use lowercase commands (e.g., `get`, `create`) to get JSON-formatted responses
- **MySQL-style tabular format**: Use UPPERCASE commands (e.g., `GET`, `CREATE`) to get native MySQL-style tabular output

The command syntax remains flexible for both formats.

#### Database and Table Selection

1. Show all available databases:
   ```
   GET dbs
   ```

2. Show all tables in the current database:
   ```
   GET tables
   ```

3. Switch to a database:
   ```
   USE database_name
   ```

4. Select a table for operations:
   ```
   USE table_name
   ```

The command prompt changes to reflect your current selections:
- Default: `noqli>`
- When a database is selected: `noqli:database_name>`
- When both database and table are selected: `noqli:database_name:table_name>`

#### CRUD Operations

NoQLi supports four main CRUD commands:

#### CREATE

Add new records to the database:

```
CREATE {name: 'John Doe', email: 'john@example.com'}
```

You can add any field, even if it doesn't exist in the table yet - NoQLi will automatically create new columns as needed.

#### GET

Retrieve records from the database:

1. Get all records:
   ```
   GET
   ```

2. Get a record by ID:
   ```
   GET 5
   ```
   or
   ```
   GET {id: 5}
   ```

3. Get multiple records by ID:
   ```
   GET {id: [1, 3, 5]}
   ```

4. Get records in an ID range:
   ```
   GET {id: (1, 10)}
   ```

#### UPDATE

Update existing records:

1. Update a single record:
   ```
   UPDATE {id: 5, name: 'New Name', status: 'active'}
   ```

2. Update multiple records:
   ```
   UPDATE {id: [1, 3, 5], status: 'inactive'}
   ```

3. Update records in a range:
   ```
   UPDATE {id: (1, 10), category: 'archived'}
   ```

#### DELETE

Delete records from the database:

1. Delete a single record:
   ```
   DELETE {id: 5}
   ```

2. Delete multiple records:
   ```
   DELETE {id: [1, 3, 5]}
   ```

3. Delete records in a range:
   ```
   DELETE {id: (1, 10)}
   ```

### Special Query Syntax

NoQLi supports some special syntax features:

1. Array value assignment to multiple fields:
   ```
   CREATE {[status, category] = 'active'}
   ```
   This sets both `status` and `category` fields to 'active'.

2. Range queries with the format:
   ```
   GET {id: (start, end)}
   ```

3. Simple ID queries:
   ```
   GET 5
   ```
   This is equivalent to `GET {id: 5}`

## Technical Details

NoQLi uses:
- Go with the official MySQL driver
- Dynamic SQL query generation with parameter binding for security
- Runtime schema modification through ALTER TABLE statements
- Regular expressions for command parsing

## Limitations

- All dynamically created columns default to VARCHAR(255)
- No support for complex joins or subqueries

## Exit

To exit the application:

```
EXIT
``` 