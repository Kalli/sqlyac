package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Query struct {
	Name string
	SQL  string
}

type Config struct {
	Confirm              bool `json:"confirm"`
	ConfirmSchemaChanges bool `json:"confirm_schema_changes"`
	ConfirmUpdates       bool `json:"confirm_updates"`
}

func main() {
	var filepath string
	var queryName string
	var confirm bool

	flag.StringVar(&filepath, "file", "", "path to sql file")
	flag.StringVar(&queryName, "name", "", "name of query to extract")
	flag.BoolVar(&confirm, "confirm", false, "prompt for confirmation before executing query (overrides config)")
	flag.Parse()
	// load config
	config, err := loadConfig()
	if err != nil {
		// if config doesn't exist, use defaults
		config = &Config{
			Confirm:              false,
			ConfirmSchemaChanges: true,
			ConfirmUpdates:       true,
		}
	}

	// handle positional args too bc that's more ergonomic
	args := flag.Args()
	if filepath == "" && len(args) > 0 {
		filepath = args[0]
	}
	if queryName == "" && len(args) > 1 {
		queryName = args[1]
	}

	if filepath == "" {
		fmt.Fprintf(os.Stderr, "usage: sqlyac <filepath> [--name <queryname>]\n")
		os.Exit(0)
	}

	if !strings.HasSuffix(filepath, ".sql") {
		fmt.Fprintf(os.Stderr, "error: file must have .sql extension\n")
		os.Exit(1)
	}

	queries, variables, err := parseSQL(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing sql: %v\n", err)
		os.Exit(1)
	}

	if queryName == "" {
		// list all available queries
		fmt.Fprintf(os.Stderr, "available queries:\n")
		for _, q := range queries {
			fmt.Fprintf(os.Stderr, "  %s\n", q.Name)
		}
		return
	}

	// find and output the requested query
	for _, q := range queries {
		if q.Name == queryName {
			// interpolate variables into the query
			interpolatedSQL, err := interpolateVariables(q.SQL, variables)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error interpolating variables: %v\n", err)
				os.Exit(1)
			}

			// check if we need confirmation based on config or flag
			needsConfirm := confirm || config.Confirm ||
				(config.ConfirmSchemaChanges && containsSchemaChanges(interpolatedSQL)) ||
				(config.ConfirmUpdates && containsUpdates(interpolatedSQL))

			if needsConfirm && !confirmQuery(q.Name, interpolatedSQL) {
				fmt.Fprintf(os.Stderr, "cancelled\n")
				os.Exit(1)
			}
			fmt.Print(interpolatedSQL)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "error: query '%s' not found\n", queryName)
	os.Exit(1)
}

func parseSQL(filepath string) ([]Query, map[string]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var queries []Query
	var currentQuery *Query
	var sqlLines []string
	variables := make(map[string]string)

	scanner := bufio.NewScanner(file)
	nameRegex := regexp.MustCompile(`--\s*@name\s*(\w+)`)
	separatorRegex := regexp.MustCompile(`^---+$`)
	// Updated regex to capture quoted vs unquoted values
	variableRegex := regexp.MustCompile(`SET\s+@(\w+)\s*=\s*(.+?)(?:;|$)`)

	for scanner.Scan() {
		line := scanner.Text()

		// check for variable definitions (SET @var="value" or SET @var=value)
		if matches := variableRegex.FindStringSubmatch(strings.TrimSpace(line)); matches != nil {
			varName := matches[1]
			varValue := strings.TrimSpace(matches[2])
			// Store the value as-is, preserving quotes or lack thereof
			variables[varName] = varValue
			continue
		}

		// check if this is a separator line
		if separatorRegex.MatchString(strings.TrimSpace(line)) {
			// if we have a current query, save it
			if currentQuery != nil && currentQuery.Name != "" {
				currentQuery.SQL = strings.TrimSpace(strings.Join(sqlLines, "\n"))
				queries = append(queries, *currentQuery)
			}
			// reset for next query
			currentQuery = &Query{}
			sqlLines = []string{}
			continue
		}

		// check for @name annotation
		if matches := nameRegex.FindStringSubmatch(line); matches != nil {
			if currentQuery != nil {
				currentQuery.Name = matches[1]
			}
			continue
		}

		// skip other comment lines that aren't @name
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}

		// accumulate sql lines
		if currentQuery != nil {
			sqlLines = append(sqlLines, line)
		}
	}

	// don't forget the last query if file doesn't end with separator
	if currentQuery != nil && currentQuery.Name != "" {
		currentQuery.SQL = strings.TrimSpace(strings.Join(sqlLines, "\n"))
		queries = append(queries, *currentQuery)
	}

	return queries, variables, scanner.Err()
}

func interpolateVariables(sql string, variables map[string]string) (string, error) {
	// Match @variable_name patterns
	variableRefRegex := regexp.MustCompile(`@(\w+)`)
	
	result := variableRefRegex.ReplaceAllStringFunc(sql, func(match string) string {
		// Extract variable name from @variable_name
		varName := variableRefRegex.FindStringSubmatch(match)[1]
		
		if value, exists := variables[varName]; exists {
			// Return the value as-is (preserving original quoting)
			return value
		}
		
		// If variable not found, return the original match
		return match
	})
	
	return result, nil
}

func confirmQuery(queryName, sql string) bool {
	lines := strings.Split(sql, "\n")
	preview := strings.Join(lines[:min(5, len(lines))], "\n")
	if len(lines) > 5 {
	    preview += "\n and " + fmt.Sprintf("%d", len(lines)) + " more lines..."
	}

	fmt.Fprintf(os.Stderr, "\nquery: %s\n", queryName)
	fmt.Fprintf(os.Stderr, "%s\n", preview)
	fmt.Fprintf(os.Stderr, "\nrun this query? (y/n): ")

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".sqlyac", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	return &config, err
}

func containsSchemaChanges(sql string) bool {
	sql = strings.ToLower(sql)
	schemaKeywords := []string{
		"drop table", "drop database", "drop schema",
		"alter table", "alter database", "alter schema",
		"create table", "create database", "create schema",
		"truncate table", "truncate",
	}

	for _, keyword := range schemaKeywords {
		if strings.Contains(sql, keyword) {
			return true
		}
	}
	return false
}

func containsUpdates(sql string) bool {
	sql = strings.ToLower(sql)
	updateKeywords := []string{"update ", "delete ", "delete from", "insert"}

	for _, keyword := range updateKeywords {
		if strings.Contains(sql, keyword) {
			return true
		}
	}
	return false
}
