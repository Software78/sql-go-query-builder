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

// Raw is a literal SQL fragment. The caller is responsible for safety.
// Use only for expressions that cannot be parameterised (e.g. "NOW()", "TRUE").
type Raw struct {
	SQL  string
	Args []any
}

// ToSQL returns the literal SQL and any pre-bound arguments.
// Note: Raw args are appended as-is; indices must be managed by the caller.
func (r Raw) ToSQL(_ dialect.Dialect, idx *int) (string, []any) {
	*idx += len(r.Args)
	return r.SQL, r.Args
}

// Func represents a SQL function call: NAME(arg1, arg2, ...).
type Func struct {
	Name string
	Args []Expr
}

// ToSQL renders the function and its arguments.
func (f Func) ToSQL(d dialect.Dialect, idx *int) (string, []any) {
	parts := make([]string, len(f.Args))
	var allArgs []any
	for i, arg := range f.Args {
		sql, args := arg.ToSQL(d, idx)
		parts[i] = sql
		allArgs = append(allArgs, args...)
	}
	return fmt.Sprintf("%s(%s)", f.Name, strings.Join(parts, ", ")), allArgs
}

// Cast represents CAST(expr AS type).
type Cast struct {
	Inner    Expr
	CastType string
}

// ToSQL renders CAST(... AS type).
func (c Cast) ToSQL(d dialect.Dialect, idx *int) (string, []any) {
	sql, args := c.Inner.ToSQL(d, idx)
	return fmt.Sprintf("CAST(%s AS %s)", sql, c.CastType), args
}
