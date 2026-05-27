package expr

import (
	"testing"

	"github.com/Software78/sql-go-query-builder/dialect"
)

func TestRaw_PlaceholderRewrite(t *testing.T) {
	idx := 1
	sql, args := Raw{SQL: "LOWER(email) = ?", Args: []any{"a@b.com"}}.ToSQL(dialect.Postgres{}, &idx)
	if sql != `LOWER(email) = $2` {
		t.Errorf("got %q want LOWER(email) = $2", sql)
	}
	if len(args) != 1 || args[0] != "a@b.com" {
		t.Errorf("args: %v", args)
	}
	if idx != 2 {
		t.Errorf("idx: got %d want 2", idx)
	}
}

func TestCast_InvalidType(t *testing.T) {
	idx := 0
	sql, args := Cast{Inner: Val{V: 1}, CastType: "TEXT; DROP"}.ToSQL(dialect.Postgres{}, &idx)
	if sql != "" || args != nil {
		t.Errorf("expected empty result for invalid cast type, got %q %v", sql, args)
	}
}

func TestCast_ValidType(t *testing.T) {
	idx := 0
	sql, args := Cast{Inner: Val{V: 1}, CastType: "text"}.ToSQL(dialect.Postgres{}, &idx)
	if sql != "CAST($1 AS TEXT)" {
		t.Errorf("got %q", sql)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Errorf("args: %v", args)
	}
}

func TestFunc_InvalidName(t *testing.T) {
	idx := 0
	sql, args := Func{Name: "COUNT; DROP", Args: nil}.ToSQL(dialect.Postgres{}, &idx)
	if sql != "" || args != nil {
		t.Errorf("expected empty result for invalid func name, got %q %v", sql, args)
	}
}

