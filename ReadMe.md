# SQLYac

*like [httpyac](https://httpyac.github.io) but for sql files*

## What is this?

SQLYac lets you write multiple sql queries in a single file and execute them individually from the command line. Write your queries in an organized file, then pipe specific ones to your database tools of choice.

## why

Ever find yourself with a bunch of sql snippets scattered across files, or copying/pasting queries from your editor to terminal? SQLYac lets you:

- organize related queries in one file
- run specific queries by name
- pipe results directly to mysql, sqlite3, psql, etc.
- maintain your sql in version control with proper organization

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