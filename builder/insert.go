package builder

import (
	"fmt"
	"strings"

	"github.com/Software78/sql-query-builder/internal/dialect"
)

// ConflictAction specifies what to do on a unique constraint violation.
type ConflictAction int

const (
	conflictNone ConflictAction = iota
	ConflictDoNothing
	ConflictDoUpdate
)

// ConflictClause is returned by OnConflict to allow fluent action chaining.
type ConflictClause struct {
	b         *InsertBuilder
	col       string
	action    ConflictAction
	updateCol string
	updateVal any
}

// DoNothing sets ON CONFLICT DO NOTHING.
func (c *ConflictClause) DoNothing() *InsertBuilder {
	c.action = ConflictDoNothing
	return c.b
}

// DoUpdate sets ON CONFLICT (col) DO UPDATE SET updateCol = val.
func (c *ConflictClause) DoUpdate(col string, val any) *InsertBuilder {
	c.action = ConflictDoUpdate
	c.updateCol = col
	c.updateVal = val
	return c.b
}

// InsertBuilder constructs an INSERT statement.
// It is NOT goroutine-safe.
type InsertBuilder struct {
	d         dialect.Dialect
	table     string
	cols      []string
	rows      [][]any
	fromSel   *SelectBuilder
	conflict  *ConflictClause
	returning []string
}

func newInsert(d dialect.Dialect) *InsertBuilder {
	return &InsertBuilder{d: d}
}

// Insert sets the target table.
func (b *InsertBuilder) Insert(table string) *InsertBuilder {
	b.table = table
	return b
}

// Columns sets the column list.
func (b *InsertBuilder) Columns(cols ...string) *InsertBuilder {
	b.cols = append(b.cols, cols...)
	return b
}

// Values adds a single row of values.
func (b *InsertBuilder) Values(vals ...any) *InsertBuilder {
	b.rows = append(b.rows, vals)
	return b
}

// BulkValues adds multiple rows at once.
func (b *InsertBuilder) BulkValues(rows [][]any) *InsertBuilder {
	b.rows = append(b.rows, rows...)
	return b
}

// FromSelect uses a SELECT statement as the source (INSERT INTO ... SELECT).
func (b *InsertBuilder) FromSelect(sel *SelectBuilder) *InsertBuilder {
	b.fromSel = sel
	return b
}

// OnConflict begins an ON CONFLICT clause for the given column.
func (b *InsertBuilder) OnConflict(col string) *ConflictClause {
	b.conflict = &ConflictClause{b: b, col: col}
	return b.conflict
}

// Returning adds a RETURNING clause.
func (b *InsertBuilder) Returning(cols ...string) *InsertBuilder {
	b.returning = append(b.returning, cols...)
	return b
}

// ToSQL renders the INSERT statement.
func (b *InsertBuilder) ToSQL() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrNoTable
	}
	if len(b.cols) == 0 && b.fromSel == nil {
		return "", nil, ErrNoColumns
	}
	if b.fromSel == nil && len(b.rows) == 0 {
		return "", nil, ErrNoValues
	}

	idx := 0
	var sb strings.Builder
	var allArgs []any

	sb.WriteString("INSERT INTO ")
	sb.WriteString(b.d.QuoteIdentifier(b.table))

	// Column list
	if len(b.cols) > 0 {
		quoted := make([]string, len(b.cols))
		for i, c := range b.cols {
			quoted[i] = b.d.QuoteIdentifier(c)
		}
		sb.WriteString(" (")
		sb.WriteString(strings.Join(quoted, ", "))
		sb.WriteByte(')')
	}

	if b.fromSel != nil {
		// INSERT INTO ... SELECT
		selSQL, selArgs, err := b.fromSel.ToSQL()
		if err != nil {
			return "", nil, fmt.Errorf("qb: INSERT FROM SELECT: %w", err)
		}
		sb.WriteByte('\n')
		sb.WriteString(selSQL)
		allArgs = append(allArgs, selArgs...)
	} else {
		// Validate column/value alignment.
		for rowIdx, row := range b.rows {
			if len(b.cols) > 0 && len(row) != len(b.cols) {
				return "", nil, fmt.Errorf("%w: row %d has %d values for %d columns",
					ErrMismatch, rowIdx, len(row), len(b.cols))
			}
		}

		// VALUES (...)
		rowParts := make([]string, len(b.rows))
		for i, row := range b.rows {
			phs := make([]string, len(row))
			for j, v := range row {
				idx++
				phs[j] = b.d.Placeholder(idx)
				allArgs = append(allArgs, v)
			}
			rowParts[i] = "(" + strings.Join(phs, ", ") + ")"
		}
		sb.WriteString("\nVALUES ")
		sb.WriteString(strings.Join(rowParts, ", "))
	}

	// ON CONFLICT
	if b.conflict != nil {
		switch b.conflict.action {
		case ConflictDoNothing:
			if b.conflict.col != "" {
				sb.WriteString("\nON CONFLICT (")
				sb.WriteString(b.d.QuoteIdentifier(b.conflict.col))
				sb.WriteString(") DO NOTHING")
			} else {
				sb.WriteString("\nON CONFLICT DO NOTHING")
			}
		case ConflictDoUpdate:
			idx++
			sb.WriteString("\nON CONFLICT (")
			sb.WriteString(b.d.QuoteIdentifier(b.conflict.col))
			sb.WriteString(") DO UPDATE SET ")
			sb.WriteString(b.d.QuoteIdentifier(b.conflict.updateCol))
			sb.WriteString(" = ")
			sb.WriteString(b.d.Placeholder(idx))
			allArgs = append(allArgs, b.conflict.updateVal)
		}
	}

	// RETURNING
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

// NewInsertBuilder creates a new InsertBuilder with the given dialect.
func NewInsertBuilder(d dialect.Dialect) *InsertBuilder {
	return newInsert(d)
}
