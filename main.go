package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Query struct {
	Name string
	SQL  string
}

func main() {
	var filepath string
	var queryName string
	
	flag.StringVar(&filepath, "file", "", "path to sql file")
	flag.StringVar(&queryName, "name", "", "name of query to extract")
	flag.Parse()

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

	queries, err := parseSQL(filepath)
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
			fmt.Print(q.SQL)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "error: query '%s' not found\n", queryName)
	os.Exit(1)
}

func parseSQL(filepath string) ([]Query, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var queries []Query
	var currentQuery *Query
	var sqlLines []string
	
	scanner := bufio.NewScanner(file)
	nameRegex := regexp.MustCompile(`--\s*@name\s*(\w+)`)
	separatorRegex := regexp.MustCompile(`^---+$`)

	for scanner.Scan() {
		line := scanner.Text()
		
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

	return queries, scanner.Err()
}