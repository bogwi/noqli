# NoQLi

NoQLi (pronounced "no-klee") is an interactive MySQL command-line interface with a flexible query syntax that combines the simplicity of NoSQL-style commands with the power of a relational database.

> **Project Status:** NoQLi is under development. The core functionality is stable and ready for use, with new features being added more or less regularly but in packs of 2-3 features at a time. Contributions and feedback are welcome!

## Features

- Interactive command-line interface for MySQL databases
- Simplified CRUD operations (CREATE, GET, UPDATE, DELETE)
- Support for complex queries including arrays and ranges
- Dynamic schema modification - automatically creates columns when needed
- Flexible query syntax with intuitive object notation
- Database and table selection commands with dynamic prompt
- Works with any MySQL database and table
- Dual output format: colorized JSON or MySQL-style tabular format
- Enhanced keyboard navigation with arrow keys (LEFT/RIGHT for editing, UP/DOWN for history)
- Namespace-aware command history (per database and table context)

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
- **Colorized JSON format**: Use lowercase commands (e.g., `get`, `create`) to get colorized JSON-formatted responses
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

5. Get records by any column:
   ```
   GET {email: 'alice@example.com'}
   ```

6. Get records by multiple columns:
   ```
   GET {name: 'Alice', status: 'active'}
   ```

7. Get records with array values:
   ```
   GET {status: ['active', 'pending']}
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

4. Update records filtered by any column:
   ```
   UPDATE {email: ['alice@example.com'], status: 'active'}
   ```
   This updates the status to 'active' for records with email 'alice@example.com'.

5. Update records filtered by array values:
   ```
   UPDATE {status: ['pending', 'review'], category: 'urgent'}
   ```
   This updates the category to 'urgent' for all records with status 'pending' or 'review'.

6. Update all records (with confirmation prompt):
   ```
   UPDATE {category: 'general'}
   ```
   This updates all records in the table after confirming with the user.

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

### Keyboard Navigation

NoQLi provides enhanced command-line editing capabilities:

- **Left/Right Arrow Keys**: Navigate through the current command to edit any part of it
- **Up/Down Arrow Keys**: Browse through command history specific to your current context
- **Tab Key**: Auto-complete common commands like USE, CREATE, GET, UPDATE, DELETE
- **Ctrl+C**: Abort the current command input
- **Ctrl+D**: Exit the application

### Command History

NoQLi maintains separate command histories for:

1. Global context (when no database or table is selected)
2. Database-specific context (when a database is selected but no table)
3. Table-specific context (when both database and table are selected)

This means that when you switch between databases or tables, your command history will be specific to that context, making it easier to recall relevant commands.

Command history is saved between sessions in `~/.noqli/history.txt`.

## Technical Details

NoQLi uses:
- Go with the official MySQL driver
- Dynamic SQL query generation with parameter binding for security
- Runtime schema modification through ALTER TABLE statements
- Regular expressions for command parsing
- Colorized JSON output via go-prettyjson
- Enhanced terminal input with line editing via liner

## Limitations

- All dynamically created columns default to VARCHAR(255)
- No support for complex joins or subqueries

## Exit

To exit the application:

```
EXIT
```

You can also press Ctrl+D to exit. 

## License

NoQLi is licensed under the Apache License, Version 2.0 (the "License");
you may not use this software except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. 