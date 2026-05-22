package builder

import (
	"strings"

	"github.com/Software78/sql-query-builder/internal/clause"
	"github.com/Software78/sql-query-builder/internal/dialect"
)

// DeleteBuilder constructs a DELETE statement.
// It is NOT goroutine-safe.
type DeleteBuilder struct {
	d         dialect.Dialect
	table     string
	using     string
	where     *clause.WhereClause
	returning []string
}

func newDelete(d dialect.Dialect) *DeleteBuilder {
	return &DeleteBuilder{
		d:     d,
		where: clause.NewWhereClause(),
	}
}

// Delete sets the target table.
func (b *DeleteBuilder) Delete(table string) *DeleteBuilder {
	b.table = table
	return b
}

// Using adds a USING clause (PostgreSQL DELETE ... USING).
func (b *DeleteBuilder) Using(table string) *DeleteBuilder {
	b.using = table
	return b
}

// Where adds a predicate joined with AND.
func (b *DeleteBuilder) Where(col, op string, val any) *DeleteBuilder {
	b.where.And(&clause.SimplePredicate{Col: col, Op: op, Val: val})
	return b
}

// WhereRaw adds a raw predicate joined with AND.
func (b *DeleteBuilder) WhereRaw(sql string, args ...any) *DeleteBuilder {
	b.where.And(&clause.RawPredicate{SQL: sql, Args: args})
	return b
}

// WhereIn adds a col IN (...) predicate.
func (b *DeleteBuilder) WhereIn(col string, vals ...any) *DeleteBuilder {
	b.where.And(&clause.InPredicate{Col: col, Vals: vals})
	return b
}

// Returning adds a RETURNING clause (PostgreSQL).
func (b *DeleteBuilder) Returning(cols ...string) *DeleteBuilder {
	b.returning = append(b.returning, cols...)
	return b
}

// ToSQL renders the DELETE statement.
func (b *DeleteBuilder) ToSQL() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrNoTable
	}

	idx := 0
	var sb strings.Builder
	var allArgs []any

	sb.WriteString("DELETE FROM ")
	sb.WriteString(b.d.QuoteIdentifier(b.table))

	if b.using != "" {
		sb.WriteString("\nUSING ")
		sb.WriteString(b.d.QuoteIdentifier(b.using))
	}

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

// NewDeleteBuilder creates a new DeleteBuilder with the given dialect.
func NewDeleteBuilder(d dialect.Dialect) *DeleteBuilder {
	return newDelete(d)
}
