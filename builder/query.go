// Package builder provides the public query builder types.
package builder

import "errors"

// Queryable is implemented by all builder types. ToSQL renders the final SQL
// string and its positional arguments. The idx parameter tracks the global
// placeholder counter; pass a pointer to a zero-value int for top-level calls.
type Queryable interface {
	ToSQL() (query string, args []any, err error)
}

// Sentinel errors returned by builder ToSQL methods.
var (
	ErrNoTable        = errors.New("qb: no table specified")
	ErrNoColumns      = errors.New("qb: no columns specified")
	ErrNoValues       = errors.New("qb: no values specified")
	ErrMismatch       = errors.New("qb: column and value count mismatch")
	ErrInvalidOp      = errors.New("qb: invalid SQL operator")
	ErrEmptyIN        = errors.New("qb: IN list must not be empty")
	ErrNilSubquery       = errors.New("qb: subquery Queryable must not be nil")
	ErrNoAlias           = errors.New("qb: subquery FROM requires an alias")
	ErrInvalidIdentifier = errors.New("qb: invalid SQL identifier")
)

// Direction is the ORDER BY sort direction.
type Direction string

const (
	ASC  Direction = "ASC"
	DESC Direction = "DESC"
)
