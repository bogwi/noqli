# NoQLi

NoQLi (pronounced "no-klee") is an interactive MySQL command-line interface with a flexible query syntax that combines the simplicity of NoSQL-style commands with the power of a relational database.

> **Project Status:** NoQLi is under development. The core functionality is stable and ready for use, with new features being added more or less regularly but in packs of 2-3 features at a time. Before any tagged release, noqli's command syntax is subject to change. But you get a glimpse of it trying even now! Be sure to check the [MySQL Coverage](MySQLcoverage.md) for the latest syntax and ideas around the corner. Contributions and feedback are welcome!

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

## Basic Usage
Make sure your mysql is running, then, after building noqli, run it as `./bin/noqli`. If all is correct, you should see the noqli prompt.

```bash
Connected to MySQL
NoQLi CLI. Type EXIT to quit.
noqli:mysql> 
```
Then run:
```bash
# this lists all databases on your mysql server
noqli:mysql> get dbs 
```
```bash
Databases: [
  "information_schema",
  "mysql",
  "performance_schema",
  "sys",
  "tutorial_db"
]
```
Get notice the command is in lowercase. This is `noqli`'s way of distinguishing between `noqli`'s commands and mysql's commands. Mysql's returns are in uppercase when using the `noqli` command line interface.
```bash
# this lists all tables in the current database using mysql's return format
noqli:mysql> GET dbs 
```
```bash
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
| tutorial_db        |

5 rows in set
```
Same is true for every command `noqli` supports.
Also notice how the command prompt changes to reflect the current database and table.
```bash
noqli:mysql> use mysql
Switched to database 'mysql'
noqli:mysql> 
```
Inspect the database and its tables with:
```bash
noqli:mysql> get tables
Tables in mysql: [
  "columns_priv",
  "component",
  "db",
  "default_roles",
  "engine_cost",
   ...
  "time_zone",
  "time_zone_leap_second",
  "time_zone_name",
  "time_zone_transition",
  "time_zone_transition_type",
  "user"
]
noqli:mysql> use engine_cost
Switched to database 'engine_cost'
noqli:engine_cost> get
Records: [
  {
    "comment": null,
    "cost_name": "io_block_read_cost",
    "cost_value": null,
    "default_value": "1",
    "device_type": "0",
    "engine_name": "default",
    "last_update": "2025-04-28 23:42:02"
  },
  {
    "comment": null,
    "cost_name": "memory_block_read_cost",
    "cost_value": null,
    "default_value": "0.25",
    "device_type": "0",
    "engine_name": "default",
    "last_update": "2025-04-28 23:42:02"
  }
]
```
As you can see, using `noqli` is very simple. Native json styled returns are unbreakable. Inspect any table with `noqli`, trace problematic entries.

At the moment it is best to use `noqli` for general inspection and use mysql's CLI for any serious work. NoQLi does support CREATE, UPDATE, DELETE, but it is too soon to introduce them into the mix.

Check out the [MySQL Coverage](MySQLcoverage.md) for the latest syntax and ideas around the corner.


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

### All

```
make all
```

## Usage

Start the CLI:

```
./bin/noqli
```

#### Output Formats

NoQLi now supports two output formats:
- **Colorized JSON format**: Use lowercase commands (e.g., `get`, `create`) to get colorized JSON-formatted responses
- **MySQL-style tabular format**: Use UPPERCASE commands (e.g., `GET`, `CREATE`) to get native MySQL-style tabular output


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