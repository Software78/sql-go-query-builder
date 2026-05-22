// Package clause provides the predicate tree used to construct WHERE and HAVING clauses.
// Predicates compose recursively, producing correctly parenthesised SQL.
package clause

import (
	"fmt"
	"strings"

	"github.com/Software78/sql-go-query-builder/internal/dialect"
)

// LogicalOp is AND or OR.
type LogicalOp string

const (
	OpAND LogicalOp = "AND"
	OpOR  LogicalOp = "OR"
)

// Predicate is a single node in the predicate tree.
type Predicate interface {
	toSQL(d dialect.Dialect, idx *int) (string, []any)
}

// WhereClause holds a top-level predicate tree and renders the WHERE keyword.
type WhereClause struct {
	root Predicate
}

// NewWhereClause creates an empty WHERE clause.
func NewWhereClause() *WhereClause { return &WhereClause{} }

// IsEmpty reports whether no predicates have been added.
func (w *WhereClause) IsEmpty() bool { return w.root == nil }

// And appends a predicate joined with AND.
func (w *WhereClause) And(p Predicate) {
	w.root = merge(w.root, p, OpAND)
}

// Or appends a predicate joined with OR.
func (w *WhereClause) Or(p Predicate) {
	w.root = merge(w.root, p, OpOR)
}

// ToSQL renders "WHERE ..." or "" if empty.
func (w *WhereClause) ToSQL(d dialect.Dialect, idx *int) (string, []any) {
	if w.root == nil {
		return "", nil
	}
	sql, args := w.root.toSQL(d, idx)
	return "WHERE " + sql, args
}

// Clone returns a deep copy of the WhereClause.
func (w *WhereClause) Clone() *WhereClause {
	return &WhereClause{root: clonePredicate(w.root)}
}

func clonePredicate(p Predicate) Predicate {
	if p == nil {
		return nil
	}
	switch v := p.(type) {
	case *CompoundPredicate:
		children := make([]Predicate, len(v.Children))
		for i, c := range v.Children {
			children[i] = clonePredicate(c)
		}
		return &CompoundPredicate{Op: v.Op, Children: children, Grouped: v.Grouped}
	default:
		return p
	}
}

func merge(existing, incoming Predicate, op LogicalOp) Predicate {
	if existing == nil {
		return incoming
	}
	if c, ok := existing.(*CompoundPredicate); ok && c.Op == op {
		c.Children = append(c.Children, incoming)
		return c
	}
	return &CompoundPredicate{Op: op, Children: []Predicate{existing, incoming}}
}

// --- Predicate implementations ---

// CompoundPredicate groups children with AND/OR.
type CompoundPredicate struct {
	Op       LogicalOp
	Children []Predicate
	Grouped  bool // wraps in parentheses when true
}

func (c *CompoundPredicate) toSQL(d dialect.Dialect, idx *int) (string, []any) {
	parts := make([]string, len(c.Children))
	var allArgs []any
	for i, child := range c.Children {
		sql, args := child.toSQL(d, idx)
		parts[i] = sql
		allArgs = append(allArgs, args...)
	}
	joined := strings.Join(parts, " "+string(c.Op)+" ")
	if c.Grouped {
		return "(" + joined + ")", allArgs
	}
	return joined, allArgs
}

// SimplePredicate handles col OP val comparisons.
type SimplePredicate struct {
	Col string
	Op  string
	Val any
}

var validOps = map[string]bool{
	"=": true, "!=": true, "<": true, "<=": true, ">": true, ">=": true,
	"LIKE": true, "NOT LIKE": true, "ILIKE": true, "NOT ILIKE": true,
	"@>": true, "<@": true, "?": true,
}

// ValidOp reports whether the operator is in the allowed set.
func ValidOp(op string) bool { return validOps[op] }

func (s *SimplePredicate) toSQL(d dialect.Dialect, idx *int) (string, []any) {
	*idx++
	col := quoteCol(d, s.Col)
	ph := d.Placeholder(*idx)
	return fmt.Sprintf("%s %s %s", col, s.Op, ph), []any{s.Val}
}

// InPredicate handles col IN (...) / NOT IN (...).
type InPredicate struct {
	Col  string
	Vals []any
	Not  bool
}

func (p *InPredicate) toSQL(d dialect.Dialect, idx *int) (string, []any) {
	placeholders := make([]string, len(p.Vals))
	for i := range p.Vals {
		*idx++
		placeholders[i] = d.Placeholder(*idx)
	}
	op := "IN"
	if p.Not {
		op = "NOT IN"
	}
	col := quoteCol(d, p.Col)
	return fmt.Sprintf("%s %s (%s)", col, op, strings.Join(placeholders, ", ")), p.Vals
}

// NullPredicate handles IS NULL / IS NOT NULL.
type NullPredicate struct {
	Col string
	Not bool
}

func (p *NullPredicate) toSQL(d dialect.Dialect, _ *int) (string, []any) {
	col := quoteCol(d, p.Col)
	if p.Not {
		return col + " IS NOT NULL", nil
	}
	return col + " IS NULL", nil
}

// BetweenPredicate handles col BETWEEN low AND high.
type BetweenPredicate struct {
	Col        string
	Low, High  any
}

func (p *BetweenPredicate) toSQL(d dialect.Dialect, idx *int) (string, []any) {
	*idx++
	lo := d.Placeholder(*idx)
	*idx++
	hi := d.Placeholder(*idx)
	col := quoteCol(d, p.Col)
	return fmt.Sprintf("%s BETWEEN %s AND %s", col, lo, hi), []any{p.Low, p.High}
}

// RawPredicate passes through a literal SQL fragment.
// Use ? as a positional placeholder token regardless of dialect;
// they will be rewritten to the correct dialect placeholder at render time.
// Use ?? for a literal ? (e.g. PostgreSQL JSONB ? operator).
// Example: WhereRaw("LOWER(email) = ?", "foo@example.com")
// Example: WhereRaw("metadata ?? ?", key)  // metadata ? $1
type RawPredicate struct {
	SQL  string
	Args []any
}

func (p *RawPredicate) toSQL(d dialect.Dialect, idx *int) (string, []any) {
	var b strings.Builder
	runes := []rune(p.SQL)
	argIdx := 0
	for i := 0; i < len(runes); i++ {
		if runes[i] == '?' {
			if i+1 < len(runes) && runes[i+1] == '?' {
				b.WriteRune('?')
				i++
				continue
			}
			if argIdx < len(p.Args) {
				*idx++
				b.WriteString(d.Placeholder(*idx))
				argIdx++
				continue
			}
		}
		b.WriteRune(runes[i])
	}
	return b.String(), p.Args
}

// quoteCol quotes a simple column name. Dot-separated "table.col" is also handled.
func quoteCol(d dialect.Dialect, col string) string {
	parts := strings.SplitN(col, ".", 2)
	if len(parts) == 2 {
		return d.QuoteIdentifier(parts[0]) + "." + d.QuoteIdentifier(parts[1])
	}
	return d.QuoteIdentifier(col)
}

// GroupedPredicate wraps an existing WhereClause as a parenthesised child predicate.
// Used by SelectBuilder.WhereGroup to nest conditions.
type GroupedPredicate struct {
	Inner *WhereClause
}

func (g *GroupedPredicate) toSQL(d dialect.Dialect, idx *int) (string, []any) {
	sql, args := g.Inner.ToSQL(d, idx)
	sql = strings.TrimPrefix(sql, "WHERE ")
	return "(" + sql + ")", args
}
