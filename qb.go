// Package qb is a production-grade, database-agnostic SQL query builder for Go.
//
// It targets PostgreSQL by default with clean dialect abstraction for MySQL.
// Builders are NOT goroutine-safe; use SelectBuilder.Clone to branch a shared base.
//
// Quick start:
//
//	pg := dialect.Postgres{}
//	q := qb.New(pg)
//
//	sql, args, err := q.Select("id", "email").
//	    From("users").
//	    Where("status", "=", "active").
//	    OrderBy("created_at", builder.DESC).
//	    Limit(20).
//	    ToSQL()
package qb

import (
	"github.com/Software78/sql-go-query-builder/builder"
	"github.com/Software78/sql-go-query-builder/dialect"
)

// QB is the root factory that creates query builders scoped to a dialect.
type QB struct {
	d dialect.Dialect
}

// New creates a QB bound to the given dialect.
// Prefer NewMySQL or NewPostgres when using this module as a dependency.
func New(d dialect.Dialect) *QB {
	return &QB{d: d}
}

// NewMySQL creates a QB for MySQL (? placeholders, backtick identifiers).
func NewMySQL() *QB {
	return New(dialect.MySQL{})
}

// NewPostgres creates a QB for PostgreSQL ($N placeholders, double-quoted identifiers).
func NewPostgres() *QB {
	return New(dialect.Postgres{})
}

// Select starts a SELECT builder with the given columns.
// Pass "*" or no arguments to select all columns.
func (q *QB) Select(cols ...string) *builder.SelectBuilder {
	sb := builder.NewSelectBuilder(q.d)
	if len(cols) > 0 {
		sb.Select(cols...)
	}
	return sb
}

// Insert starts an INSERT builder for the given table.
func (q *QB) Insert(table string) *builder.InsertBuilder {
	return builder.NewInsertBuilder(q.d).Insert(table)
}

// Update starts an UPDATE builder for the given table.
func (q *QB) Update(table string) *builder.UpdateBuilder {
	return builder.NewUpdateBuilder(q.d).Update(table)
}

// Delete starts a DELETE builder for the given table.
func (q *QB) Delete(table string) *builder.DeleteBuilder {
	return builder.NewDeleteBuilder(q.d).Delete(table)
}
