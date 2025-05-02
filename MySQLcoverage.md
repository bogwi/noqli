# MySQL to NoQLi Command Coverage

This document tracks the mapping between standard MySQL commands and their NoQLi equivalents. A checkmark (✅) indicates the MySQL command is supported by NoQLi.

## Data Manipulation

| MySQL Command | NoQLi Equivalent | Supported |
|---------------|------------------|-----------|
| `SELECT * FROM table` | `GET` | ✅ |
| `SELECT * FROM table WHERE id = 5` | `GET 5` or `GET {id: 5}` | ✅ |
| `SELECT * FROM table WHERE col = 'value'` | `GET {col: 'value'}` | ✅ |
| `SELECT * FROM table WHERE col IN ('val1', 'val2')` | `GET {col: ['val1', 'val2']}` | ✅ |
| `SELECT * FROM table WHERE id BETWEEN 1 AND 10` | `GET {id: (1, 10)}` | ✅ |
| `SELECT column1, column2 FROM table_name` | `GET {column1, column2}` | ✅  |
| `SELECT * FROM table WHERE col1 = 'val1' AND col2 = 'val2'` | `GET {col1: 'val1', col2: 'val2'}` | ✅ |
| `SELECT * FROM table ORDER BY col` | `GET {UP: 'col'}` | ✅ |
| `SELECT * FROM table ORDER BY col DESC` | `GET {DOWN: 'col'}` | ✅ |
| `SELECT * FROM table LIMIT 10` | `GET {LIM: 10}` | ✅ |
| `SELECT * FROM table LIMIT 10 OFFSET 20` | `GET {LIM: 10, OFF: 20}` | ✅ |
| `SELECT * FROM table WHERE col LIKE '%pattern%'` | `GET {LIKE: 'pattern'}` | ✅ |
| `SELECT COUNT(*) FROM table` | `GET {COUNT: '*'}` | ✅  |
| `SELECT COUNT(email) FROM table` | `GET {COUNT: 'email'}` | ✅  |
| `SELECT COUNT(DISTINCT email) FROM table` | `GET {COUNT: 'email', DISTINCT: true}` | ✅  |
| `SELECT COUNT(*) FROM table WHERE country = 'USA'` | `GET {COUNT: '*', country: 'USA'}` | ✅  |
| `SELECT MIN(col) FROM table` | `GET {MIN: 'col'}` | ✅ |
| `SELECT MAX(col) FROM table` | `GET {MAX: 'col'}` | ✅ |
| `SELECT AVG(col) FROM table` | `GET {AVG: 'col'}` | ✅ |
| `SELECT SUM(col) FROM table` | `GET {SUM: 'col'}` | ✅ |
| `INSERT INTO table (col1, col2) VALUES ('val1', 'val2')` | `CREATE {col1: 'val1', col2: 'val2'}` | ✅ |
| `UPDATE table SET col = 'value' WHERE id = 5` | `UPDATE {id: 5, col: 'value'}` | ✅ |
| `UPDATE table SET col = 'value' WHERE id IN (1, 3, 5)` | `UPDATE {id: [1, 3, 5], col: 'value'}` | ✅ |
| `UPDATE table SET col = 'value' WHERE id BETWEEN 1 AND 10` | `UPDATE {id: (1, 10), col: 'value'}` | ✅ |
| `DELETE FROM table WHERE id = 5` | `DELETE {id: 5}` | ✅ |
| `DELETE FROM table WHERE id IN (1, 3, 5)` | `DELETE {id: [1, 3, 5]}` | ✅ |
| `DELETE FROM table WHERE id BETWEEN 1 AND 10` | `DELETE {id: (1, 10)}` | ✅ |


## Schema Manipulation

| MySQL Command | NoQLi Equivalent | Supported |
|---------------|------------------|-----------|
| `SHOW DATABASES` | `GET dbs` | ✅ |
| `SHOW TABLES` | `GET tables` | ✅ |
| `USE database_name` | `USE database_name` | ✅ |
| `USE table_name` | `USE table_name` | ✅ |
| `DESCRIBE table` or `SHOW COLUMNS FROM table` | `GET schema` | ❌ |
| `CREATE DATABASE db_name` | `MAKE DB db_name` | ❌ |
| `CREATE TABLE table_name (...)` | `MAKE TABLE table_name (...)` | ❌ |
| `ALTER TABLE table ADD COLUMN col VARCHAR(255)` | *Auto-created when needed* | ✅ |
| `ALTER TABLE table DROP COLUMN col` | `DROP: {'col'}` | ❌ |
| `ALTER TABLE table RENAME TO new_table` | `RENAME TABLE new_table` | ❌ |
| `DROP TABLE table` | `DROP table_name` | ❌ |
| `DROP DATABASE db_name` | `DROP db_name` | ❌ |

## JOINs and Advanced Queries

| MySQL Command | NoQLi Equivalent | Supported |
|---------------|------------------|-----------|
| `SELECT * FROM table1 JOIN table2 ON table1.col = table2.col` | `JOIN {table1: 'col', table2: 'col'}` | ❌ |
| `SELECT * FROM table1 LEFT JOIN table2 ON table1.col = table2.col` | `LEFT JOIN {table1: 'col', table2: 'col'}` | ❌ |
| `SELECT * FROM table1 RIGHT JOIN table2 ON table1.col = table2.col` | `RIGHT JOIN {table1: 'col', table2: 'col'}` | ❌ |
| `SELECT * FROM table WHERE col1 = 'val1' OR col2 = 'val2'` | `GET {OR: [{col1: 'val1'}, {col2: 'val2'}]}` | ❌ |
| `SELECT * FROM table GROUP BY col` | `GET {GROUP BY: 'col'}` | ❌ |
| `SELECT * FROM table HAVING col > value` | `GET {HAVING: {col: '> value'}}` | ❌ |
| `SELECT * FROM (SELECT * FROM table) AS subquery` | Not supported | ❌ |

## Transactions and Access Control

| MySQL Command | NoQLi Equivalent | Supported |
|---------------|------------------|-----------|
| `START TRANSACTION` | `BEGIN` | ❌ |
| `COMMIT` | `COMMIT` | ❌ |
| `ROLLBACK` | `ROLLBACK` | ❌ |
| `GRANT privileges ON db.table TO user` | Not supported | ❌ |
| `REVOKE privileges ON db.table FROM user` | Not supported | ❌ |
| `CREATE USER user IDENTIFIED BY 'password'` | Not supported | ❌ |
| `DROP USER user` | Not supported | ❌ |

## Indexes and Performance

| MySQL Command | NoQLi Equivalent | Supported |
|---------------|------------------|-----------|
| `CREATE INDEX idx ON table (col)` | `CREATE INDEX {table: 'col'}` | ❌ |
| `DROP INDEX idx ON table` | `DROP INDEX {table: 'idx'}` | ❌ |
| `EXPLAIN SELECT * FROM table` | `EXPLAIN` | ❌ |
| `ANALYZE TABLE table` | Not supported | ❌ |
| `OPTIMIZE TABLE table` | Not supported | ❌ |

## Proposed New NoQLi Commands

For the unsupported MySQL commands, here are proposed NoQLi syntax extensions that would align with the existing NoQLi philosophy:

1. **Ordering and Limiting**:
   ```
   GET {UP: 'column_name'} 
   GET {DOWN: 'column_name'}
   GET {LIM: 10}
   GET {LIM: 10, OFF: 20}
   ```

2. **Pattern Matching**:
   ```
   GET {LIKE: 'pattern'}  // LIKE pattern matching
   ```

3. **Aggregation Functions**:
   ```
   GET {COUNT: '*'} 
   GET {MAX: 'column_name'}
   GET {AVG: 'column_name'}
   GET {SUM: 'column_name'}
   GET {column: 'value', GROUP BY: 'group_column'}
   ```

4. **Schema Operations**:
   ```
   GET schema  // Show table structure
   CREATE DATABASE db_name
   CREATE TABLE {column1: 'VARCHAR(255)', column2: 'INT'}
   ALTER TABLE {DROP: 'column_name'}
   RENAME TABLE new_table_name
   DROP TABLE
   DROP DB db_name
   ```

5. **Joins**:
   ```
   JOIN {table1: 'column', table2: 'column'}
   LEFT JOIN {table1: 'column', table2: 'column'}
   RIGHT JOIN {table1: 'column', table2: 'column'}
   ```

6. **Complex Conditions**:
   ```
   GET {OR: [{column1: 'value1'}, {column2: 'value2'}]}
   ```

7. **Transactions**:
   ```
   BEGIN
   COMMIT
   ROLLBACK
   ```

8. **Indexes**:
   ```
   CREATE INDEX {table: 'column_name'}
   DROP INDEX {table: 'index_name'}
   EXPLAIN GET {column: 'value'}
   ```

## Implementation Priority

Based on common MySQL usage patterns, here's a suggested implementation priority for the missing commands:

1. Aggregation functions
2. Schema inspection
3. Joins for basic relationship queries
4. Transactions
5. Indexes and performance optimization

These extensions would significantly enhance NoQLi's coverage of common MySQL functionality while maintaining its simplified, intuitive syntax philosophy.