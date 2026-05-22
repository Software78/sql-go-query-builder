package builder

import (
	"fmt"
	"strings"

	"github.com/Software78/sql-go-query-builder/internal/clause"
	"github.com/Software78/sql-go-query-builder/internal/dialect"
)
// joinType classifies a JOIN variant.
type joinType string

const (
	joinInner joinType = "JOIN"
	joinLeft  joinType = "LEFT JOIN"
	joinRight joinType = "RIGHT JOIN"
	joinCross joinType = "CROSS JOIN"
)

type join struct {
	typ       joinType
	table     string
	condition string // empty for CROSS JOIN
}

type orderItem struct {
	col string
	dir Direction
}

type cte struct {
	name string
	sub  Queryable
}

// SelectBuilder constructs a SELECT statement.
// It is NOT goroutine-safe; use Clone to branch off a shared base.
type SelectBuilder struct {
	d          dialect.Dialect
	cols       []string
	fromTable  string
	fromSub    Queryable
	fromAlias  string
	joins      []join
	where      *clause.WhereClause
	groupBy    []string
	having     *clause.WhereClause
	orderBy    []orderItem
	limit      int64
	offset     int64
	forUpdate  bool
	ctes       []cte
	err        error
}

func newSelect(d dialect.Dialect) *SelectBuilder {
	return &SelectBuilder{
		d:      d,
		limit:  -1,
		offset: -1,
		where:  clause.NewWhereClause(),
		having: clause.NewWhereClause(),
	}
}

// Clone returns a deep copy so the original builder is not mutated.
func (s *SelectBuilder) Clone() *SelectBuilder {
	cp := *s
	cp.cols = append([]string(nil), s.cols...)
	cp.joins = append([]join(nil), s.joins...)
	cp.groupBy = append([]string(nil), s.groupBy...)
	cp.orderBy = append([]orderItem(nil), s.orderBy...)
	cp.ctes = append([]cte(nil), s.ctes...)
	cp.where = s.where.Clone()
	cp.having = s.having.Clone()
	return &cp
}

// Select sets the columns to retrieve. Pass "*" for all columns.
func (s *SelectBuilder) Select(cols ...string) *SelectBuilder {
	s.cols = append(s.cols, cols...)
	return s
}

// From sets the table name.
func (s *SelectBuilder) From(table string) *SelectBuilder {
	s.fromTable = table
	return s
}

// FromSubquery sets a subquery as the FROM source with the given alias.
func (s *SelectBuilder) FromSubquery(sub Queryable, alias string) *SelectBuilder {
	if sub == nil {
		s.err = ErrNilSubquery
		return s
	}
	s.fromSub = sub
	s.fromAlias = alias
	return s
}

// Join adds an INNER JOIN.
func (s *SelectBuilder) Join(table, condition string) *SelectBuilder {
	s.joins = append(s.joins, join{joinInner, table, condition})
	return s
}

// LeftJoin adds a LEFT JOIN.
func (s *SelectBuilder) LeftJoin(table, condition string) *SelectBuilder {
	s.joins = append(s.joins, join{joinLeft, table, condition})
	return s
}

// RightJoin adds a RIGHT JOIN.
func (s *SelectBuilder) RightJoin(table, condition string) *SelectBuilder {
	s.joins = append(s.joins, join{joinRight, table, condition})
	return s
}

// CrossJoin adds a CROSS JOIN.
func (s *SelectBuilder) CrossJoin(table string) *SelectBuilder {
	s.joins = append(s.joins, join{joinCross, table, ""})
	return s
}

// Where adds a col OP val predicate joined with AND.
func (s *SelectBuilder) Where(col, op string, val any) *SelectBuilder {
	if !clause.ValidOp(op) {
		s.err = fmt.Errorf("%w: %q", ErrInvalidOp, op)
		return s
	}
	s.where.And(&clause.SimplePredicate{Col: col, Op: op, Val: val})
	return s
}

// WhereRaw adds a raw SQL predicate joined with AND.
// Use ? as a positional placeholder token; it is rewritten to the dialect placeholder at render time.
func (s *SelectBuilder) WhereRaw(sql string, args ...any) *SelectBuilder {
	s.where.And(&clause.RawPredicate{SQL: sql, Args: args})
	return s
}

// WhereIn adds a col IN (...) predicate.
func (s *SelectBuilder) WhereIn(col string, vals ...any) *SelectBuilder {
	if len(vals) == 0 {
		s.err = ErrEmptyIN
		return s
	}
	s.where.And(&clause.InPredicate{Col: col, Vals: vals, Not: false})
	return s
}

// WhereNotIn adds a col NOT IN (...) predicate.
func (s *SelectBuilder) WhereNotIn(col string, vals ...any) *SelectBuilder {
	s.where.And(&clause.InPredicate{Col: col, Vals: vals, Not: true})
	return s
}

// WhereNull adds a col IS NULL predicate.
func (s *SelectBuilder) WhereNull(col string) *SelectBuilder {
	s.where.And(&clause.NullPredicate{Col: col, Not: false})
	return s
}

// WhereNotNull adds a col IS NOT NULL predicate.
func (s *SelectBuilder) WhereNotNull(col string) *SelectBuilder {
	s.where.And(&clause.NullPredicate{Col: col, Not: true})
	return s
}

// WhereBetween adds a col BETWEEN low AND high predicate.
func (s *SelectBuilder) WhereBetween(col string, low, high any) *SelectBuilder {
	s.where.And(&clause.BetweenPredicate{Col: col, Low: low, High: high})
	return s
}

// OrWhere adds a col OP val predicate joined with OR.
func (s *SelectBuilder) OrWhere(col, op string, val any) *SelectBuilder {
	if !clause.ValidOp(op) {
		s.err = fmt.Errorf("%w: %q", ErrInvalidOp, op)
		return s
	}
	s.where.Or(&clause.SimplePredicate{Col: col, Op: op, Val: val})
	return s
}

// WhereGroup adds a grouped (parenthesised) set of predicates joined with AND.
// The callback receives a fresh builder; any Where* calls on it are collected
// and wrapped in parentheses as a single nested predicate.
func (s *SelectBuilder) WhereGroup(fn func(b *SelectBuilder)) *SelectBuilder {
	inner := newSelect(s.d)
	fn(inner)
	if inner.where.IsEmpty() {
		return s
	}
	s.where.And(&clause.GroupedPredicate{Inner: inner.where})
	return s
}

// GroupBy adds GROUP BY columns.
func (s *SelectBuilder) GroupBy(cols ...string) *SelectBuilder {
	s.groupBy = append(s.groupBy, cols...)
	return s
}

// Having adds a HAVING predicate.
func (s *SelectBuilder) Having(col, op string, val any) *SelectBuilder {
	if !clause.ValidOp(op) {
		s.err = fmt.Errorf("%w: %q", ErrInvalidOp, op)
		return s
	}
	s.having.And(&clause.SimplePredicate{Col: col, Op: op, Val: val})
	return s
}

// OrderBy adds an ORDER BY column with direction.
func (s *SelectBuilder) OrderBy(col string, dir Direction) *SelectBuilder {
	s.orderBy = append(s.orderBy, orderItem{col, dir})
	return s
}

// Limit sets the LIMIT clause. Pass -1 to unset.
func (s *SelectBuilder) Limit(n int64) *SelectBuilder {
	s.limit = n
	return s
}

// Offset sets the OFFSET clause.
func (s *SelectBuilder) Offset(n int64) *SelectBuilder {
	s.offset = n
	return s
}

// ForUpdate appends FOR UPDATE to the query.
func (s *SelectBuilder) ForUpdate() *SelectBuilder {
	s.forUpdate = true
	return s
}

// WithCTE prepends a Common Table Expression (WITH name AS (sub)).
func (s *SelectBuilder) WithCTE(name string, sub Queryable) *SelectBuilder {
	if sub == nil {
		s.err = fmt.Errorf("%w: CTE %q", ErrNilSubquery, name)
		return s
	}
	s.ctes = append(s.ctes, cte{name, sub})
	return s
}

// ToSQL renders the SELECT statement and its positional arguments.
func (s *SelectBuilder) ToSQL() (string, []any, error) {
	if s.err != nil {
		return "", nil, s.err
	}
	if s.fromTable == "" && s.fromSub == nil {
		return "", nil, ErrNoTable
	}

	idx := 0
	var sb strings.Builder
	var allArgs []any

	// CTEs
	if len(s.ctes) > 0 {
		sb.WriteString("WITH ")
		for i, c := range s.ctes {
			if i > 0 {
				sb.WriteString(", ")
			}
			sql, args, err := c.sub.ToSQL()
			if err != nil {
				return "", nil, fmt.Errorf("qb: CTE %q: %w", c.name, err)
			}
			sb.WriteString(s.d.QuoteIdentifier(c.name))
			sb.WriteString(" AS (")
			sb.WriteString(sql)
			sb.WriteByte(')')
			allArgs = append(allArgs, args...)
			idx += len(args)
		}
		sb.WriteByte('\n')
	}

	// SELECT
	sb.WriteString("SELECT ")
	if len(s.cols) == 0 {
		sb.WriteByte('*')
	} else {
		quoted := make([]string, len(s.cols))
		for i, c := range s.cols {
			if c == "*" || strings.ContainsAny(c, ".()*") {
				quoted[i] = c
			} else {
				quoted[i] = s.d.QuoteIdentifier(c)
			}
		}
		sb.WriteString(strings.Join(quoted, ", "))
	}

	// FROM
	sb.WriteString("\nFROM ")
	if s.fromSub != nil {
		if s.fromAlias == "" {
			return "", nil, ErrNoAlias
		}
		subSQL, subArgs, err := s.fromSub.ToSQL()
		if err != nil {
			return "", nil, fmt.Errorf("qb: FROM subquery: %w", err)
		}
		sb.WriteByte('(')
		sb.WriteString(subSQL)
		sb.WriteString(") ")
		sb.WriteString(s.d.QuoteIdentifier(s.fromAlias))
		allArgs = append(allArgs, subArgs...)
		idx += len(subArgs)
	} else {
		sb.WriteString(s.d.QuoteIdentifier(s.fromTable))
	}

	// JOINs
	for _, j := range s.joins {
		sb.WriteByte('\n')
		sb.WriteString(string(j.typ))
		sb.WriteByte(' ')
		sb.WriteString(s.d.QuoteIdentifier(j.table))
		if j.condition != "" {
			sb.WriteString(" ON ")
			sb.WriteString(j.condition)
		}
	}

	// WHERE
	if !s.where.IsEmpty() {
		wherSQL, whereArgs := s.where.ToSQL(s.d, &idx)
		sb.WriteByte('\n')
		sb.WriteString(wherSQL)
		allArgs = append(allArgs, whereArgs...)
	}

	// GROUP BY
	if len(s.groupBy) > 0 {
		quoted := make([]string, len(s.groupBy))
		for i, c := range s.groupBy {
			quoted[i] = s.d.QuoteIdentifier(c)
		}
		sb.WriteString("\nGROUP BY ")
		sb.WriteString(strings.Join(quoted, ", "))
	}

	// HAVING
	if !s.having.IsEmpty() {
		havSQL, havArgs := s.having.ToSQL(s.d, &idx)
		havSQL = strings.Replace(havSQL, "WHERE ", "HAVING ", 1)
		sb.WriteByte('\n')
		sb.WriteString(havSQL)
		allArgs = append(allArgs, havArgs...)
	}

	// ORDER BY
	if len(s.orderBy) > 0 {
		sb.WriteString("\nORDER BY ")
		parts := make([]string, len(s.orderBy))
		for i, o := range s.orderBy {
			parts[i] = s.d.QuoteIdentifier(o.col) + " " + string(o.dir)
		}
		sb.WriteString(strings.Join(parts, ", "))
	}

	// LIMIT / OFFSET
	if s.limit >= 0 || s.offset > 0 {
		lim := s.limit
		off := s.offset
		if off < 0 {
			off = 0
		}
		limitSQL := s.d.LimitClause(lim, off)
		if limitSQL != "" {
			sb.WriteByte('\n')
			sb.WriteString(limitSQL)
		}
	}

	// FOR UPDATE
	if s.forUpdate {
		sb.WriteString("\nFOR UPDATE")
	}

	return sb.String(), allArgs, nil
}

// Ensure SelectBuilder satisfies Queryable.
var _ Queryable = (*SelectBuilder)(nil)

// NewSelectBuilder creates a new SelectBuilder with the given dialect.
//
// Deprecated: Use QB.Select via qb.New, qb.NewPostgres, or qb.NewMySQL instead.
// This constructor is retained only for advanced embedding use cases.
func NewSelectBuilder(d dialect.Dialect) *SelectBuilder {
	return newSelect(d)
}
