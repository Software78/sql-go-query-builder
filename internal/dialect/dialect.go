// Package dialect defines the SQL dialect abstraction used by the query builder.
// Each dialect controls placeholder style, identifier quoting, and LIMIT syntax.
package dialect

import (
	"fmt"
	"strings"
)

// Dialect abstracts database-specific SQL syntax differences.
type Dialect interface {
	// Placeholder returns the parameter placeholder for the Nth argument (1-indexed).
	Placeholder(n int) string

	// QuoteIdentifier wraps an identifier (table/column name) in dialect-appropriate quotes.
	QuoteIdentifier(s string) string

	// LimitClause returns the LIMIT/OFFSET SQL fragment. Pass -1 to omit either.
	LimitClause(limit, offset int64) string

	// Name returns a human-readable name for the dialect (e.g. "postgres", "mysql").
	Name() string
}

// Postgres is the PostgreSQL dialect.
type Postgres struct{}

// Placeholder returns $N style placeholders used by PostgreSQL.
func (Postgres) Placeholder(n int) string { return fmt.Sprintf("$%d", n) }

// QuoteIdentifier double-quotes the identifier, escaping any internal double quotes.
func (Postgres) QuoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// LimitClause returns LIMIT N OFFSET M. Pass -1 to omit either term.
func (Postgres) LimitClause(limit, offset int64) string {
	return DefaultLimitClause(limit, offset)
}

// Name returns "postgres".
func (Postgres) Name() string { return "postgres" }

// MySQL is the MySQL dialect.
type MySQL struct{}

// Placeholder returns ? style placeholders used by MySQL.
func (MySQL) Placeholder(_ int) string { return "?" }

// QuoteIdentifier backtick-quotes the identifier, escaping internal backticks.
func (MySQL) QuoteIdentifier(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

// LimitClause returns LIMIT N OFFSET M. Pass -1 to omit either term.
func (MySQL) LimitClause(limit, offset int64) string {
	return DefaultLimitClause(limit, offset)
}

// DefaultLimitClause is the shared LIMIT/OFFSET implementation for dialects
// that use standard SQL syntax. Pass -1 to omit either term.
func DefaultLimitClause(limit, offset int64) string {
	var b strings.Builder
	if limit >= 0 {
		fmt.Fprintf(&b, "LIMIT %d", limit)
	}
	if offset > 0 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "OFFSET %d", offset)
	}
	return b.String()
}

// Name returns "mysql".
func (MySQL) Name() string { return "mysql" }
