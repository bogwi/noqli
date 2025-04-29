# NoQLi Go Installation Guide

This guide provides step-by-step instructions for installing the Go version of NoQLi.

## Prerequisites

- A MySQL database server
- Internet connection for downloading Go and dependencies

## Installation Steps

### 1. Install Go

If you don't have Go installed, you can:

- Use the provided setup script:
  ```
  ./setup.sh
  ```
  This script will detect your operating system and help install Go.

- Or manually install Go:
  - Download from [golang.org/dl](https://golang.org/dl/)
  - Follow the installation instructions for your operating system

### 2. Configure Database

1. Create a MySQL database for NoQLi:
   ```sql
   CREATE DATABASE noqli;
   ```

2. Create a users table:
   ```sql
   USE noqli;
   CREATE TABLE users (
     id INT AUTO_INCREMENT PRIMARY KEY
   );
   ```

3. Create a MySQL user (optional, you can use an existing user):
   ```sql
   CREATE USER 'noqli_user'@'localhost' IDENTIFIED BY 'your_password';
   GRANT ALL PRIVILEGES ON noqli.* TO 'noqli_user'@'localhost';
   FLUSH PRIVILEGES;
   ```

### 3. Set Up NoQLi

1. Create a `.env` file with your database credentials:
   ```
   cp env.example .env
   ```

2. Edit the `.env` file with your database information:
   ```
   DB_HOST=localhost
   DB_USER=noqli_user
   DB_PASSWORD=your_password
   DB_NAME=noqli
   ```

3. Install Go dependencies:
   ```
   go mod tidy
   ```

4. Build the application:
   ```
   go build -o noqli
   ```

### 4. Run NoQLi

Start the application:
```
./noqli
```

You should see:
```
Connected to MySQL
Direct MySQL CLI. Type EXIT to quit.
> 
```

Now you can start using the NoQLi commands as described in the README.md file.

## Troubleshooting

1. **Connection errors**: Make sure your MySQL server is running and that the credentials in the `.env` file are correct.

2. **Permission issues**: Ensure the MySQL user has sufficient privileges on the database.

3. **Go installation problems**: If the setup script fails, try installing Go manually from the official website. 