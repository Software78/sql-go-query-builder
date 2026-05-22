// Package expr provides SQL expression primitives used throughout the query builder.
// All Expr implementations produce SQL fragments and bound arguments.
package expr

import (
	"fmt"
	"strings"

	"github.com/Software78/sql-go-query-builder/internal/dialect"
)

// Expr is a self-contained SQL expression that can render itself with the given
// dialect and a shared placeholder index counter.
type Expr interface {
	ToSQL(d dialect.Dialect, idx *int) (sql string, args []any)
}

// Col is a quoted column reference, optionally table-qualified: "table"."col".
type Col struct {
	Table string // optional
	Name  string
}

// ToSQL renders the column reference as a quoted identifier.
func (c Col) ToSQL(d dialect.Dialect, _ *int) (string, []any) {
	if c.Table != "" {
		return d.QuoteIdentifier(c.Table) + "." + d.QuoteIdentifier(c.Name), nil
	}
	return d.QuoteIdentifier(c.Name), nil
}

// Val is a bound parameter value. It consumes one placeholder index slot.
type Val struct {
	V any
}

// ToSQL renders a placeholder and appends the value to args.
func (v Val) ToSQL(d dialect.Dialect, idx *int) (string, []any) {
	*idx++
	return d.Placeholder(*idx), []any{v.V}
}

// Raw is a literal SQL fragment. Use ? as a positional placeholder token;
// it is rewritten to the dialect placeholder at render time.
// Use only for expressions that cannot be parameterised (e.g. "NOW()", "TRUE").
type Raw struct {
	SQL  string
	Args []any
}

// ToSQL returns the literal SQL and bound arguments with placeholders rewritten.
func (r Raw) ToSQL(d dialect.Dialect, idx *int) (string, []any) {
	var b strings.Builder
	argIdx := 0
	for _, ch := range r.SQL {
		if ch == '?' && argIdx < len(r.Args) {
			*idx++
			b.WriteString(d.Placeholder(*idx))
			argIdx++
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String(), r.Args
}

// Func represents a SQL function call: NAME(arg1, arg2, ...).
type Func struct {
	Name string
	Args []Expr
}

// ToSQL renders the function and its arguments.
func (f Func) ToSQL(d dialect.Dialect, idx *int) (string, []any) {
	if !isFuncName(f.Name) {
		return "", nil
	}
	parts := make([]string, len(f.Args))
	var allArgs []any
	for i, arg := range f.Args {
		sql, args := arg.ToSQL(d, idx)
		parts[i] = sql
		allArgs = append(allArgs, args...)
	}
	return fmt.Sprintf("%s(%s)", f.Name, strings.Join(parts, ", ")), allArgs
}

func isFuncName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
		} else if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// Cast represents CAST(expr AS type).
type Cast struct {
	Inner    Expr
	CastType string
}

var validCastTypes = map[string]bool{
	"INT": true, "INTEGER": true, "BIGINT": true, "SMALLINT": true,
	"TEXT": true, "VARCHAR": true, "BOOLEAN": true, "BOOL": true,
	"FLOAT": true, "NUMERIC": true, "DECIMAL": true,
	"DATE": true, "TIMESTAMP": true, "TIMESTAMPTZ": true, "UUID": true,
	"JSONB": true, "JSON": true,
}

// ToSQL renders CAST(... AS type). CastType must be in the allowlist.
func (c Cast) ToSQL(d dialect.Dialect, idx *int) (string, []any) {
	upper := strings.ToUpper(c.CastType)
	if !validCastTypes[upper] {
		return "", nil
	}
	sql, args := c.Inner.ToSQL(d, idx)
	return fmt.Sprintf("CAST(%s AS %s)", sql, upper), args
}
