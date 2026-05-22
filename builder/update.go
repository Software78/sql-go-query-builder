package builder

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Software78/sql-go-query-builder/internal/clause"
	"github.com/Software78/sql-go-query-builder/internal/dialect"
	"github.com/Software78/sql-go-query-builder/internal/expr"
)

type setItem struct {
	col    string
	val    any
	expr   expr.Expr
	isExpr bool
}

// UpdateBuilder constructs an UPDATE statement.
// It is NOT goroutine-safe.
type UpdateBuilder struct {
	d         dialect.Dialect
	table     string
	sets      []setItem
	where     *clause.WhereClause
	returning []string
	err       error
}

func newUpdate(d dialect.Dialect) *UpdateBuilder {
	return &UpdateBuilder{
		d:     d,
		where: clause.NewWhereClause(),
	}
}

// Update sets the target table.
func (b *UpdateBuilder) Update(table string) *UpdateBuilder {
	b.table = table
	return b
}

// Set adds a col = $N assignment.
func (b *UpdateBuilder) Set(col string, val any) *UpdateBuilder {
	b.sets = append(b.sets, setItem{col: col, val: val})
	return b
}

// SetExpr sets col = expression using a typed expr.Expr (e.g. expr.Raw{SQL: "stock - 1"}).
// Prefer this over SetRaw for non-parameterised right-hand sides.
func (b *UpdateBuilder) SetExpr(col string, e expr.Expr) *UpdateBuilder {
	b.sets = append(b.sets, setItem{col: col, expr: e, isExpr: true})
	return b
}

// SetRaw adds a col = raw_expr assignment (e.g. "stock - 1").
//
// Deprecated: use SetExpr with expr.Raw instead. The expression is inserted verbatim;
// caller is responsible for safety — never pass user-controlled strings here.
func (b *UpdateBuilder) SetRaw(col, rawExpr string) *UpdateBuilder {
	return b.SetExpr(col, expr.Raw{SQL: rawExpr})
}

// SetMap adds multiple col = val assignments from a map.
// Columns are sorted alphabetically to guarantee deterministic SQL output.
func (b *UpdateBuilder) SetMap(m map[string]any) *UpdateBuilder {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.sets = append(b.sets, setItem{col: k, val: m[k]})
	}
	return b
}

// Where adds a predicate joined with AND.
func (b *UpdateBuilder) Where(col, op string, val any) *UpdateBuilder {
	if !clause.ValidOp(op) {
		b.err = fmt.Errorf("%w: %q", ErrInvalidOp, op)
		return b
	}
	b.where.And(&clause.SimplePredicate{Col: col, Op: op, Val: val})
	return b
}

// WhereRaw adds a raw predicate joined with AND.
func (b *UpdateBuilder) WhereRaw(sql string, args ...any) *UpdateBuilder {
	b.where.And(&clause.RawPredicate{SQL: sql, Args: args})
	return b
}

// WhereIn adds a col IN (...) predicate.
func (b *UpdateBuilder) WhereIn(col string, vals ...any) *UpdateBuilder {
	if len(vals) == 0 {
		b.err = ErrEmptyIN
		return b
	}
	b.where.And(&clause.InPredicate{Col: col, Vals: vals})
	return b
}

// WhereNull adds a col IS NULL predicate.
func (b *UpdateBuilder) WhereNull(col string) *UpdateBuilder {
	b.where.And(&clause.NullPredicate{Col: col, Not: false})
	return b
}

// WhereNotNull adds a col IS NOT NULL predicate.
func (b *UpdateBuilder) WhereNotNull(col string) *UpdateBuilder {
	b.where.And(&clause.NullPredicate{Col: col, Not: true})
	return b
}

// OrWhere adds a col OP val predicate joined with OR.
func (b *UpdateBuilder) OrWhere(col, op string, val any) *UpdateBuilder {
	if !clause.ValidOp(op) {
		b.err = fmt.Errorf("%w: %q", ErrInvalidOp, op)
		return b
	}
	b.where.Or(&clause.SimplePredicate{Col: col, Op: op, Val: val})
	return b
}

// WhereGroup adds a grouped (parenthesised) set of predicates joined with AND.
// The callback receives a fresh UpdateBuilder; any Where* calls on it are
// collected and wrapped in parentheses as a single nested predicate.
func (b *UpdateBuilder) WhereGroup(fn func(b *UpdateBuilder)) *UpdateBuilder {
	inner := newUpdate(b.d)
	fn(inner)
	if inner.err != nil {
		b.err = inner.err
		return b
	}
	if inner.where.IsEmpty() {
		return b
	}
	b.where.And(&clause.GroupedPredicate{Inner: inner.where})
	return b
}

// Returning adds a RETURNING clause (PostgreSQL).
func (b *UpdateBuilder) Returning(cols ...string) *UpdateBuilder {
	b.returning = append(b.returning, cols...)
	return b
}

// ToSQL renders the UPDATE statement.
func (b *UpdateBuilder) ToSQL() (string, []any, error) {
	if b.err != nil {
		return "", nil, b.err
	}
	if b.table == "" {
		return "", nil, ErrNoTable
	}
	if len(b.sets) == 0 {
		return "", nil, ErrNoColumns
	}

	idx := 0
	var sb strings.Builder
	var allArgs []any

	sb.WriteString("UPDATE ")
	sb.WriteString(b.d.QuoteIdentifier(b.table))
	sb.WriteString("\nSET ")

	setParts := make([]string, len(b.sets))
	for i, s := range b.sets {
		col := b.d.QuoteIdentifier(s.col)
		if s.isExpr {
			exprSQL, exprArgs := s.expr.ToSQL(b.d, &idx)
			setParts[i] = col + " = " + exprSQL
			allArgs = append(allArgs, exprArgs...)
		} else {
			idx++
			setParts[i] = col + " = " + b.d.Placeholder(idx)
			allArgs = append(allArgs, s.val)
		}
	}
	sb.WriteString(strings.Join(setParts, ", "))

	if !b.where.IsEmpty() {
		wherSQL, whereArgs := b.where.ToSQL(b.d, &idx)
		sb.WriteByte('\n')
		sb.WriteString(wherSQL)
		allArgs = append(allArgs, whereArgs...)
	}

	if len(b.returning) > 0 {
		quoted := make([]string, len(b.returning))
		for i, c := range b.returning {
			quoted[i] = b.d.QuoteIdentifier(c)
		}
		sb.WriteString("\nRETURNING ")
		sb.WriteString(strings.Join(quoted, ", "))
	}

	return sb.String(), allArgs, nil
}

// NewUpdateBuilder creates a new UpdateBuilder with the given dialect.
func NewUpdateBuilder(d dialect.Dialect) *UpdateBuilder {
	return newUpdate(d)
}
