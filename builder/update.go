package builder

import (
	"strings"

	"github.com/Software78/sql-query-builder/internal/clause"
	"github.com/Software78/sql-query-builder/internal/dialect"
)

type setItem struct {
	col    string
	val    any
	rawVal string // non-empty means raw expression
	isRaw  bool
}

// UpdateBuilder constructs an UPDATE statement.
// It is NOT goroutine-safe.
type UpdateBuilder struct {
	d         dialect.Dialect
	table     string
	sets      []setItem
	where     *clause.WhereClause
	returning []string
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

// SetRaw adds a col = raw_expr assignment (e.g. "quantity = quantity + 1").
// The expression is inserted verbatim — caller is responsible for safety.
func (b *UpdateBuilder) SetRaw(col, expr string) *UpdateBuilder {
	b.sets = append(b.sets, setItem{col: col, rawVal: expr, isRaw: true})
	return b
}

// SetMap adds multiple col = val assignments from a map.
func (b *UpdateBuilder) SetMap(m map[string]any) *UpdateBuilder {
	for k, v := range m {
		b.sets = append(b.sets, setItem{col: k, val: v})
	}
	return b
}

// Where adds a predicate joined with AND.
func (b *UpdateBuilder) Where(col, op string, val any) *UpdateBuilder {
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
	b.where.And(&clause.InPredicate{Col: col, Vals: vals})
	return b
}

// Returning adds a RETURNING clause (PostgreSQL).
func (b *UpdateBuilder) Returning(cols ...string) *UpdateBuilder {
	b.returning = append(b.returning, cols...)
	return b
}

// ToSQL renders the UPDATE statement.
func (b *UpdateBuilder) ToSQL() (string, []any, error) {
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
		if s.isRaw {
			setParts[i] = col + " = " + s.rawVal
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
