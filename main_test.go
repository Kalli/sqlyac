package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseSQL(t *testing.T) {
	// create a test sql file
	testSQL := `---
-- @name CreateUsersTable
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username VARCHAR(50) NOT NULL
);

---
-- @name InsertSampleUsers
INSERT INTO users (username) VALUES 
    ('alice'),
    ('bob');

---
-- @name GetAllUsers
SELECT * FROM users ORDER BY username;
---`

	// write to temp file
	tmpFile, err := os.CreateTemp("", "test*.sql")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testSQL); err != nil {
		t.Fatalf("failed to write test sql: %v", err)
	}
	tmpFile.Close()

	// parse the file
	queries, err := parseSQL(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}

	// verify we got the expected queries
	expectedQueries := []string{"CreateUsersTable", "InsertSampleUsers", "GetAllUsers"}
	if len(queries) != len(expectedQueries) {
		t.Errorf("expected %d queries, got %d", len(expectedQueries), len(queries))
	}

	for i, expected := range expectedQueries {
		if i >= len(queries) {
			t.Errorf("missing query: %s", expected)
			continue
		}
		if queries[i].Name != expected {
			t.Errorf("expected query name %s, got %s", expected, queries[i].Name)
		}
	}

	// verify specific query content
	createTableQuery := queries[0]
	if !strings.Contains(createTableQuery.SQL, "CREATE TABLE users") {
		t.Errorf("CreateUsersTable query doesn't contain expected SQL")
	}
	if strings.Contains(createTableQuery.SQL, "@name") {
		t.Errorf("parsed SQL should not contain @name annotation")
	}
	if strings.Contains(createTableQuery.SQL, "---") {
		t.Errorf("parsed SQL should not contain separators")
	}

	insertQuery := queries[1]
	if !strings.Contains(insertQuery.SQL, "INSERT INTO users") {
		t.Errorf("InsertSampleUsers query doesn't contain expected SQL")
	}
	if !strings.Contains(insertQuery.SQL, "alice") {
		t.Errorf("InsertSampleUsers query should contain sample data")
	}
}

func TestParseSQLWithWhitespace(t *testing.T) {
	// test with various whitespace patterns
	testSQL := `---
--    @name   QueryWithSpaces   
SELECT * FROM test;

---
--@nameNoSpaces
SELECT 1;

---
-- @name QueryWithExtraLines

SELECT * 
FROM users 
WHERE active = 1;

---`

	tmpFile, err := os.CreateTemp("", "whitespace*.sql")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(testSQL)
	tmpFile.Close()

	queries, err := parseSQL(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}

	if len(queries) != 3 {
		t.Errorf("expected 3 queries, got %d", len(queries))
	}

	// check that names are parsed correctly despite whitespace
	expectedNames := []string{"QueryWithSpaces", "NoSpaces", "QueryWithExtraLines"}
	for i, expected := range expectedNames {
		if i >= len(queries) {
			t.Errorf("missing query: %s", expected)
			continue
		}
		if queries[i].Name != expected {
			t.Errorf("expected query name %s, got %s", expected, queries[i].Name)
		}
	}

	// check that extra whitespace is trimmed from SQL
	lastQuery := queries[2]
	if strings.HasPrefix(lastQuery.SQL, "\n") || strings.HasSuffix(lastQuery.SQL, "\n\n") {
		t.Errorf("SQL should have leading/trailing whitespace trimmed: %q", lastQuery.SQL)
	}
}

func TestParseSQLEmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "empty*.sql")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	queries, err := parseSQL(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseSQL failed on empty file: %v", err)
	}

	if len(queries) != 0 {
		t.Errorf("expected 0 queries from empty file, got %d", len(queries))
	}
}

func TestParseSQLMissingFile(t *testing.T) {
	_, err := parseSQL("nonexistent.sql")
	if err == nil {
		t.Error("expected error for missing file, got none")
	}
}

func TestParseSQLWithComments(t *testing.T) {
	testSQL := `---
-- @name QueryWithComments
-- This is a comment that should be ignored
SELECT * FROM users; -- inline comment
-- Another comment
WHERE active = 1;
`

	tmpFile, err := os.CreateTemp("", "comments*.sql")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(testSQL)
	tmpFile.Close()

	queries, err := parseSQL(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}

	if len(queries) != 1 {
		t.Errorf("expected 1 query, got %d", len(queries))
	}

	query := queries[0]
	if query.Name != "QueryWithComments" {
		t.Errorf("expected query name QueryWithComments, got %s", query.Name)
	}

	// the sql should contain the actual query but not the comment-only lines
	if !strings.Contains(query.SQL, "SELECT * FROM users") {
		t.Errorf("query should contain SELECT statement")
	}
	if !strings.Contains(query.SQL, "WHERE active = 1") {
		t.Errorf("query should contain WHERE clause")
	}
	// but comment-only lines should be filtered out
	lines := strings.Split(query.SQL, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && strings.HasPrefix(trimmed, "-- ") {
			t.Errorf("comment-only lines should be filtered out, found: %s", line)
		}
	}
}

func TestParseSQLNoQueryName(t *testing.T) {
	// test file with separators but no @name annotations
	testSQL := `---
SELECT * FROM users;

---
SELECT * FROM orders;
---`

	tmpFile, err := os.CreateTemp("", "noname*.sql")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(testSQL)
	tmpFile.Close()

	queries, err := parseSQL(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}

	// should get 0 queries bc no @name annotations
	if len(queries) != 0 {
		t.Errorf("expected 0 queries without @name annotations, got %d", len(queries))
	}
}

// integration test using the actual example.sql structure
func TestExampleSQLStructure(t *testing.T) {
	// simulate the structure from example.sql
	exampleQueries := []string{
		"CreateUsersTable",
		"CreateOrdersTable", 
		"InsertSampleUsers",
		"InsertSampleOrders",
		"GetAllUsers",
		"GetActiveUsers",
		"GetLargeOrders",
		"GetUserOrderSummary",
		"GetRecentOrders",
		"CountOrdersByStatus",
		"CleanupTestData",
	}

	queries, err := parseSQL("example.sql")
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}

	if len(queries) != len(exampleQueries) {
		t.Errorf("expected %d queries, got %d", len(exampleQueries), len(queries))
	}

	// verify all expected query names are present
	queryNames := make(map[string]bool)
	for _, q := range queries {
		queryNames[q.Name] = true
	}

	for _, expected := range exampleQueries {
		if !queryNames[expected] {
			t.Errorf("missing expected query: %s", expected)
		}
	}
}