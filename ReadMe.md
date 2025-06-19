# SQLYac

*like [httpyac](https://httpyac.github.io) but for sql files*

## What is SQLYac?

SQLYac lets you write multiple sql queries in a single file and execute them individually from the command line. Write your queries in an organized file, then pipe specific ones to your database tools of choice.

## Why?

Ever find yourself with a bunch of sql snippets scattered across files, copying/pasting queries from your editor to the terminal or sql shell? SQLYac lets you:

- Organize related queries in one file
- Run specific queries by name
- Pipe results directly to mysql, sqlite3, psql, etc.
- Maintain your sql in version control with proper organization
- Add confirmation before running potentially destructive queries (see the config section below)

## Installation

```bash
go install github.com/kalli/sqlyac
# or just `go run main.go` if you're feeling it
```

## Usage

```bash
# list all available queries
sqlyac example.sql

# run a specific query
sqlyac example.sql QueryName | mysql -u user -p database

# with flags (same thing)
sqlyac --file example.sql --name QueryName | sqlite3 db.sqlite
```

## File format

Use three dashes (`---`) as separators between queries, annotate your queries with `@name`. Example:

```sql
---
-- @name GetActiveUsers
SELECT user_id, username, last_login 
FROM Users 
WHERE active = 1
ORDER BY last_login DESC;
---

---
-- @name GetLargeOrders
SELECT order_id, customer_id, total_amount
FROM Orders 
WHERE total_amount > 1000
AND created_at > DATE_SUB(NOW(), INTERVAL 30 DAY);
---
```

## Variables

SQLYac supports variables for reusable values across queries. Define variables using `SET @variable_name="value"` syntax anywhere in your file, then reference them in queries using `@variable_name`. Here's an example:

```sql
-- @name QueryWithVariables
SELECT * FROM orders o, users u
WHERE u.id=@user_id 
AND u.active=@active
AND o.status=@status
LIMIT @lim;

SET @user_id=2;
SET @lim=10;
SET @active=true;
SET @status="completed";
```

When you run `sqlyac file.sql QueryWithVariables`, the output will be:

```sql
SELECT * FROM orders o, users u
WHERE u.id=2
AND u.active=true
AND o.status="completed"
LIMIT 10;
```

## Examples

Explore what's available

```bash 
$ sqlyac example.sql
available queries:
  CreateUsersTable
  CreateOrdersTable
  InsertSampleUsers
  InsertSampleOrders
  GetAllUsers
  GetActiveUsers
  GetLargeOrders
  GetUserOrderSummary
  GetRecentOrders
  CountOrdersByStatus
  CleanupTestData
```

Run a query:

```bash 
$ sqlyac analytics.sql GetActiveUsers | mysql -u admin -p ecommerce_db --table
Enter password: 
+--------+----------+---------------------+
| user_id| username | last_login          |
+--------+----------+---------------------+
|   1234 | alice    | 2024-03-15 14:30:22 |
|   5678 | bob      | 2024-03-14 09:15:11 |
+--------+----------+---------------------+

```

Pipe to file:

```bash 
$ sqlyac analytics.sql GetLargeOrders | mysql -u admin -p ecommerce_db > large_orders.csv
```

## Workflow

1. Write your queries in `.sql` files, separate them by dashes (`---`) and annotate with `@name`
2. Version control your sql alongside your code
3. Run queries directly from terminal
4. Pipe results to any database tool or file

## Config 

You can save a configuration file in `~/.sqlyac/config.json` with the following settings: 

* `confirm` - Ask for confirmation on all queries.
* `confirm_schema_changes` - Ask for confirmation on any queries that change the database schema (i.e. `drop table`, `alter table` etc).
* `confirm_updates` boolean - Ask for confirmation on any queries that create, update or delete rows.

Here's an example that would ask for confirmation on all updates, inserts and schema changes:

```json
{
    "confirm": false,
    "confirm_schema_changes": true,
    "confirm_updates": true
}
```

Running any commands with the `--confirm` toggle overrides your config and asks for confirmation every time.

## Notes

- only parses `.sql` files
- ignores comment lines (except `@name` annotations)
- strips leading/trailing whitespace from queries
- pretty forgiving with whitespace in `@name` annotations


## Tests

Run tests like so: 

```sh
go test -v
```