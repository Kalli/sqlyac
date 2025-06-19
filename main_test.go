package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"reflect"
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
	queries, _, err := parseSQL(tmpFile.Name())
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

	queries, _, err := parseSQL(tmpFile.Name())
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

	queries, _, err := parseSQL(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseSQL failed on empty file: %v", err)
	}

	if len(queries) != 0 {
		t.Errorf("expected 0 queries from empty file, got %d", len(queries))
	}
}

func TestParseSQLMissingFile(t *testing.T) {
	_, _, err := parseSQL("nonexistent.sql")
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

	queries, _, err := parseSQL(tmpFile.Name())
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

	queries, _, err := parseSQL(tmpFile.Name())
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
		"QueryWithVariables",
	}

	queries, _, err := parseSQL("example.sql")
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


func TestLoadConfig(t *testing.T) {
	// create a temp config file
	tempDir, err := os.MkdirTemp("", "sqlyac_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configDir := filepath.Join(tempDir, ".sqlyac")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	testConfig := Config{
		Confirm:              true,
		ConfirmSchemaChanges: false,
		ConfirmUpdates:       true,
	}

	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// temporarily override home dir for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// test loading config
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if config.Confirm != testConfig.Confirm {
		t.Errorf("expected Confirm %v, got %v", testConfig.Confirm, config.Confirm)
	}
	if config.ConfirmSchemaChanges != testConfig.ConfirmSchemaChanges {
		t.Errorf("expected ConfirmSchemaChanges %v, got %v", testConfig.ConfirmSchemaChanges, config.ConfirmSchemaChanges)
	}
	if config.ConfirmUpdates != testConfig.ConfirmUpdates {
		t.Errorf("expected ConfirmUpdates %v, got %v", testConfig.ConfirmUpdates, config.ConfirmUpdates)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	// test with non-existent config file
	tempDir, err := os.MkdirTemp("", "sqlyac_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	_, err = loadConfig()
	if err == nil {
		t.Error("expected error for missing config file, got none")
	}
}

func TestContainsSchemaChanges(t *testing.T) {
	testCases := []struct {
		sql      string
		expected bool
		desc     string
	}{
		{"SELECT * FROM users", false, "basic select"},
		{"DROP TABLE users", true, "drop table"},
		{"drop table users", true, "drop table lowercase"},
		{"CREATE TABLE test (id INT)", true, "create table"},
		{"ALTER TABLE users ADD COLUMN email VARCHAR(100)", true, "alter table"},
		{"TRUNCATE TABLE logs", true, "truncate table"},
		{"INSERT INTO users VALUES (1, 'test')", false, "insert statement"},
		{"UPDATE users SET name = 'test'", false, "update statement"},
		{"DELETE FROM users WHERE id = 1", false, "delete statement"},
		{"DROP DATABASE testdb", true, "drop database"},
		{"CREATE SCHEMA analytics", true, "create schema"},
		{"-- DROP TABLE users\nSELECT * FROM users", true, "drop in comment still counts"},
	}

	for _, tc := range testCases {
		result := containsSchemaChanges(tc.sql)
		if result != tc.expected {
			t.Errorf("containsSchemaChanges(%q) = %v, expected %v (%s)", tc.sql, result, tc.expected, tc.desc)
		}
	}
}

func TestContainsUpdates(t *testing.T) {
	testCases := []struct {
		sql      string
		expected bool
		desc     string
	}{
		{"SELECT * FROM users", false, "basic select"},
		{"UPDATE users SET name = 'test'", true, "update statement"},
		{"update users set name = 'test'", true, "update lowercase"},
		{"DELETE FROM users WHERE id = 1", true, "delete statement"},
		{"delete from users where id = 1", true, "delete lowercase"},
		{"INSERT INTO users VALUES (1, 'test')", true, "insert statement"},
		{"CREATE TABLE users (id INT)", false, "create table"},
		{"DROP TABLE users", false, "drop table"},
		{"-- UPDATE users\nSELECT * FROM users", true, "update in comment still counts"},
		{"DELETE users WHERE id = 1", true, "delete without FROM"},
	}

	for _, tc := range testCases {
		result := containsUpdates(tc.sql)
		if result != tc.expected {
			t.Errorf("containsUpdates(%q) = %v, expected %v (%s)", tc.sql, result, tc.expected, tc.desc)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	// test that default behavior works when no config exists
	tempDir, err := os.MkdirTemp("", "sqlyac_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// this should fail to load config, triggering default behavior
	_, err = loadConfig()
	if err == nil {
		t.Error("expected error for missing config, got none")
	}
}

func TestConfirmationLogic(t *testing.T) {
	testCases := []struct {
		sql                  string
		confirm              bool
		configConfirm        bool
		configSchemaChanges  bool
		configUpdates        bool
		expectedConfirmation bool
		desc                 string
	}{
		{
			sql:     "SELECT * FROM users",
			confirm: false, configConfirm: false, configSchemaChanges: true, configUpdates: true,
			expectedConfirmation: false,
			desc:                 "safe query, no confirmation needed",
		},
		{
			sql:     "DROP TABLE users",
			confirm: false, configConfirm: false, configSchemaChanges: true, configUpdates: true,
			expectedConfirmation: true,
			desc:                 "schema change, should confirm",
		},
		{
			sql:     "UPDATE users SET name = 'test'",
			confirm: false, configConfirm: false, configSchemaChanges: true, configUpdates: true,
			expectedConfirmation: true,
			desc:                 "update query, should confirm",
		},
		{
			sql:     "SELECT * FROM users",
			confirm: true, configConfirm: false, configSchemaChanges: false, configUpdates: false,
			expectedConfirmation: true,
			desc:                 "flag override, should confirm",
		},
		{
			sql:     "SELECT * FROM users",
			confirm: false, configConfirm: true, configSchemaChanges: false, configUpdates: false,
			expectedConfirmation: true,
			desc:                 "config.confirm true, should confirm all",
		},
		{
			sql:     "DROP TABLE users",
			confirm: false, configConfirm: false, configSchemaChanges: false, configUpdates: true,
			expectedConfirmation: false,
			desc:                 "schema change but config disabled, no confirmation",
		},
	}

	for _, tc := range testCases {
		config := &Config{
			Confirm:              tc.configConfirm,
			ConfirmSchemaChanges: tc.configSchemaChanges,
			ConfirmUpdates:       tc.configUpdates,
		}

		// simulate the confirmation logic from main()
		needsConfirm := tc.confirm || config.Confirm ||
			(config.ConfirmSchemaChanges && containsSchemaChanges(tc.sql)) ||
			(config.ConfirmUpdates && containsUpdates(tc.sql))

		if needsConfirm != tc.expectedConfirmation {
			t.Errorf("confirmation logic failed for: %s\nsql: %q\nexpected: %v, got: %v",
				tc.desc, tc.sql, tc.expectedConfirmation, needsConfirm)
		}
	}
}

func TestParseVariables(t *testing.T) {
		// Create a temporary SQL file for testing
		content := `SET @user_id=123;
SET @status="active";
SET @limit=10;
SET @active=true;

---
-- @name SelectUser
SELECT * 
FROM Users
WHERE id=@user_id;
---

---
-- @name SelectActiveUsers
SELECT * 
FROM Users 
WHERE status=@status
LIMIT @limit;
---`

		tmpfile, err := os.CreateTemp("", "test*.sql")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		queries, variables, err := parseSQL(tmpfile.Name())
		if err != nil {
			t.Fatalf("parseSQL failed: %v", err)
		}

		expectedVariables := map[string]string{
			"status":  `"active"`, // preserves quotes
			"user_id": `123`,      // no quotes
			"limit":   `10`,       // no quotes
			"active":  `true`,     // no quotes
		}

		if !reflect.DeepEqual(variables, expectedVariables) {
			t.Errorf("Expected variables %v, got %v", expectedVariables, variables)
		}

		if len(queries) != 2 {
			t.Errorf("Expected 2 queries, got %d", len(queries))
		}

		// Test variable interpolation
		query1 := queries[0]
		interpolated1, err := interpolateVariables(query1.SQL, variables)
		if err != nil {
			t.Fatalf("interpolateVariables failed: %v", err)
		}

		expected1 := `SELECT * 
FROM Users
WHERE id=123;`

		if interpolated1 != expected1 {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected1, interpolated1)
		}

		query2 := queries[1]
		interpolated2, err := interpolateVariables(query2.SQL, variables)
		if err != nil {
			t.Fatalf("interpolateVariables failed: %v", err)
		}

		expected2 := `SELECT * 
FROM Users 
WHERE status="active"
LIMIT 10;`

		if interpolated2 != expected2 {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected2, interpolated2)
		}
}

func TestInterpolateVariablesWithMissingVar(t *testing.T) {
		sql := "SELECT * FROM Users WHERE id=@missing_var AND status=@status"
		variables := map[string]string{
			"status": `"active"`,
		}

		result, err := interpolateVariables(sql, variables)
		if err != nil {
			t.Fatalf("interpolateVariables failed: %v", err)
		}

		expected := `SELECT * FROM Users WHERE id=@missing_var AND status="active"`
		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
}

