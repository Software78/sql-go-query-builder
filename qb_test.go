package qb_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Software78/sql-go-query-builder"
	"github.com/Software78/sql-go-query-builder/builder"
	"github.com/Software78/sql-go-query-builder/internal/dialect"
	"github.com/Software78/sql-go-query-builder/internal/expr"
)

// normalise collapses all whitespace sequences to a single space so SQL
// formatting differences never cause test flakes.
func normalise(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func assertSQL(t *testing.T, got, want string) {
	t.Helper()
	if normalise(got) != normalise(want) {
		t.Errorf("\ngot:  %s\nwant: %s", normalise(got), normalise(want))
	}
}

func assertArgs(t *testing.T, got, want []any) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("args len: got %d want %d — %v vs %v", len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("args[%d]: got %v want %v", i, got[i], want[i])
		}
	}
}

var pg = qb.New(dialect.Postgres{})
var my = qb.New(dialect.MySQL{})

// ---------------------------------------------------------------------------
// SELECT
// ---------------------------------------------------------------------------

func TestSelect_Basic(t *testing.T) {
	sql, args, err := pg.Select("id", "email").From("users").ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id", "email" FROM "users"`)
	assertArgs(t, args, nil)
}

func TestSelect_Star(t *testing.T) {
	sql, _, err := pg.Select("*").From("users").ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT * FROM "users"`)
}

func TestSelect_Where(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("users").
		Where("status", "=", "active").
		Where("age", ">", 18).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "status" = $1 AND "age" > $2`)
	assertArgs(t, args, []any{"active", 18})
}

func TestSelect_OrWhere(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("users").
		Where("role", "=", "admin").
		OrWhere("role", "=", "moderator").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "role" = $1 OR "role" = $2`)
	assertArgs(t, args, []any{"admin", "moderator"})
}

func TestSelect_WhereIn(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("users").
		WhereIn("status", "active", "pending").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "status" IN ($1, $2)`)
	assertArgs(t, args, []any{"active", "pending"})
}

func TestSelect_WhereNotIn(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("orders").
		WhereNotIn("state", "cancelled", "refunded").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "orders" WHERE "state" NOT IN ($1, $2)`)
	assertArgs(t, args, []any{"cancelled", "refunded"})
}

func TestSelect_WhereNull(t *testing.T) {
	sql, _, err := pg.Select("id").From("users").WhereNull("deleted_at").ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "deleted_at" IS NULL`)
}

func TestSelect_WhereNotNull(t *testing.T) {
	sql, _, err := pg.Select("id").From("users").WhereNotNull("email").ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "email" IS NOT NULL`)
}

func TestSelect_WhereBetween(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("orders").
		WhereBetween("amount", 100, 500).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "orders" WHERE "amount" BETWEEN $1 AND $2`)
	assertArgs(t, args, []any{100, 500})
}

func TestSelect_WhereRaw(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("users").
		WhereRaw("LOWER(email) = ?", "test@example.com").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE LOWER(email) = $1`)
	assertArgs(t, args, []any{"test@example.com"})
}

func TestSelect_WhereRaw_JSONBOperator(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("docs").
		WhereRaw("metadata ?? ?", "published").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "docs" WHERE metadata ? $1`)
	assertArgs(t, args, []any{"published"})
}

func TestSelect_WhereRaw_EscapedQuestionMark(t *testing.T) {
	sql, _, err := pg.Select("id").
		From("docs").
		WhereRaw("metadata ?? 'published'").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "docs" WHERE metadata ? 'published'`)
}

func TestSelect_WhereNotIn_Empty_Error(t *testing.T) {
	_, _, err := pg.Select("id").From("orders").WhereNotIn("state").ToSQL()
	if !errors.Is(err, builder.ErrEmptyIN) {
		t.Errorf("expected ErrEmptyIN, got %v", err)
	}
}

func TestSelect_OrderBy_InvalidDirection_Error(t *testing.T) {
	_, _, err := pg.Select("id").From("users").
		OrderBy("id", builder.Direction("NULLS FIRST")).
		ToSQL()
	if err == nil {
		t.Fatal("expected error for invalid ORDER BY direction")
	}
}

func TestSelect_SafeExpression_AliasWordBoundary(t *testing.T) {
	sql, _, err := pg.Select("COUNT(*) AS selected_count").From("users").ToSQL()
	if err != nil {
		t.Fatalf("expected valid expression alias, got %v", err)
	}
	assertSQL(t, sql, `SELECT COUNT(*) AS selected_count FROM "users"`)
}

func TestSelect_WhereRaw_ComposedPlaceholderIndex(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("users").
		Where("active", "=", true).
		WhereRaw("LOWER(email) = ?", "test@example.com").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "active" = $1 AND LOWER(email) = $2`)
	assertArgs(t, args, []any{true, "test@example.com"})
}

func TestSelect_WhereGroup(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("users").
		Where("active", "=", true).
		WhereGroup(func(b *builder.SelectBuilder) {
			b.Where("role", "=", "admin").OrWhere("role", "=", "owner")
		}).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "active" = $1 AND ("role" = $2 OR "role" = $3)`)
	assertArgs(t, args, []any{true, "admin", "owner"})
}

func TestSelect_Joins(t *testing.T) {
	sql, _, err := pg.Select("u.id", "o.total").
		From("users").
		Join("orders", "orders.user_id", "users.id").
		LeftJoin("profiles", "profiles.user_id", "users.id").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	// dot-qualified column strings are passed through as-is (contain a dot)
	assertSQL(t, sql, `SELECT "u"."id", "o"."total" FROM "users" JOIN "orders" ON "orders"."user_id" = "users"."id" LEFT JOIN "profiles" ON "profiles"."user_id" = "users"."id"`)
}

func TestSelect_GroupByHaving(t *testing.T) {
	sql, args, err := pg.Select("user_id", "COUNT(*) AS cnt").
		From("orders").
		GroupBy("user_id").
		Having("cnt", ">", 5).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "user_id", COUNT(*) AS cnt FROM "orders" GROUP BY "user_id" HAVING "cnt" > $1`)
	assertArgs(t, args, []any{5})
}

func TestSelect_OrderByLimitOffset(t *testing.T) {
	sql, _, err := pg.Select("id").
		From("users").
		OrderBy("created_at", builder.DESC).
		OrderBy("id", builder.ASC).
		Limit(10).
		Offset(20).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" ORDER BY "created_at" DESC, "id" ASC LIMIT 10 OFFSET 20`)
}

func TestSelect_ForUpdate(t *testing.T) {
	sql, _, err := pg.Select("id").From("users").Where("id", "=", 1).ForUpdate().ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "id" = $1 FOR UPDATE`)
}

func TestSelect_FromSubquery(t *testing.T) {
	sub := pg.Select("id", "email").From("users").Where("active", "=", true)
	sql, args, err := pg.Select("id").FromSubquery(sub, "active_users").ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM (SELECT "id", "email" FROM "users" WHERE "active" = $1) "active_users"`)
	assertArgs(t, args, []any{true})
}

func TestSelect_CTE(t *testing.T) {
	cte := pg.Select("id", "amount").From("orders").Where("status", "=", "paid")
	sql, args, err := pg.Select("id").
		From("paid_orders").
		WithCTE("paid_orders", cte).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `WITH "paid_orders" AS (SELECT "id", "amount" FROM "orders" WHERE "status" = $1) SELECT "id" FROM "paid_orders"`)
	assertArgs(t, args, []any{"paid"})
}

func TestSelect_PlaceholderIndicesComposed(t *testing.T) {
	sql, args, err := pg.Select("id").
		From("users").
		Where("a", "=", 1).
		WhereIn("b", 10, 20, 30).
		Where("c", "=", 2).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id" FROM "users" WHERE "a" = $1 AND "b" IN ($2, $3, $4) AND "c" = $5`)
	assertArgs(t, args, []any{1, 10, 20, 30, 2})
}

func TestSelect_PlaceholderIndices_SubqueryComposed(t *testing.T) {
	sub := pg.Select("user_id").From("bans").Where("reason", "=", "spam")
	sql, args, err := pg.Select("id", "email").
		FromSubquery(sub, "banned").
		Where("active", "=", true).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT "id", "email" FROM (SELECT "user_id" FROM "bans" WHERE "reason" = $1) "banned" WHERE "active" = $2`)
	assertArgs(t, args, []any{"spam", true})
}

func TestSelect_NoTable_Error(t *testing.T) {
	_, _, err := pg.Select("id").ToSQL()
	if !errors.Is(err, builder.ErrNoTable) {
		t.Errorf("expected ErrNoTable, got %v", err)
	}
}

func TestSelect_FromSubquery_NoAlias_Error(t *testing.T) {
	sub := pg.Select("id").From("users")
	_, _, err := pg.Select("id").FromSubquery(sub, "").ToSQL()
	if !errors.Is(err, builder.ErrNoAlias) {
		t.Errorf("expected ErrNoAlias, got %v", err)
	}
}

func TestSelect_FromSubquery_Nil_Error(t *testing.T) {
	_, _, err := pg.Select("id").FromSubquery(nil, "sub").ToSQL()
	if !errors.Is(err, builder.ErrNilSubquery) {
		t.Errorf("expected ErrNilSubquery, got %v", err)
	}
}

func TestSelect_WithCTE_Nil_Error(t *testing.T) {
	_, _, err := pg.Select("id").From("users").WithCTE("x", nil).ToSQL()
	if !errors.Is(err, builder.ErrNilSubquery) {
		t.Errorf("expected ErrNilSubquery, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// INSERT
// ---------------------------------------------------------------------------

func TestInsert_SingleRow(t *testing.T) {
	sql, args, err := pg.Insert("users").
		Columns("name", "email").
		Values("Owen", "owen@example.com").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `INSERT INTO "users" ("name", "email") VALUES ($1, $2)`)
	assertArgs(t, args, []any{"Owen", "owen@example.com"})
}

func TestInsert_BulkValues(t *testing.T) {
	sql, args, err := pg.Insert("users").
		Columns("name", "email").
		BulkValues([][]any{
			{"Alice", "alice@example.com"},
			{"Bob", "bob@example.com"},
		}).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `INSERT INTO "users" ("name", "email") VALUES ($1, $2), ($3, $4)`)
	assertArgs(t, args, []any{"Alice", "alice@example.com", "Bob", "bob@example.com"})
}

func TestInsert_OnConflictDoNothing(t *testing.T) {
	sql, args, err := pg.Insert("users").
		Columns("email").
		Values("owen@example.com").
		OnConflict("email").DoNothing().
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `INSERT INTO "users" ("email") VALUES ($1) ON CONFLICT ("email") DO NOTHING`)
	assertArgs(t, args, []any{"owen@example.com"})
}

func TestInsert_OnConflictDoUpdate_Returning(t *testing.T) {
	sql, args, err := pg.Insert("users").
		Columns("email", "name").
		Values("owen@example.com", "Owen").
		OnConflict("email").DoUpdate("name", "Owen Updated").Back().
		Returning("id", "updated_at").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `INSERT INTO "users" ("email", "name") VALUES ($1, $2) ON CONFLICT ("email") DO UPDATE SET "name" = $3 RETURNING "id", "updated_at"`)
	assertArgs(t, args, []any{"owen@example.com", "Owen", "Owen Updated"})
}

func TestInsert_FromSelect(t *testing.T) {
	sel := pg.Select("name", "email").From("staging_users").Where("verified", "=", true)
	sql, args, err := pg.Insert("users").
		Columns("name", "email").
		FromSelect(sel).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `INSERT INTO "users" ("name", "email") SELECT "name", "email" FROM "staging_users" WHERE "verified" = $1`)
	assertArgs(t, args, []any{true})
}

func TestInsert_NoTable_Error(t *testing.T) {
	b := builder.NewInsertBuilder(dialect.Postgres{})
	_, _, err := b.Columns("a").Values(1).ToSQL()
	if !errors.Is(err, builder.ErrNoTable) {
		t.Errorf("expected ErrNoTable, got %v", err)
	}
}

func TestInsert_Mismatch_Error(t *testing.T) {
	_, _, err := pg.Insert("users").
		Columns("a", "b").
		Values(1). // only 1 value for 2 columns
		ToSQL()
	if !errors.Is(err, builder.ErrMismatch) {
		t.Errorf("expected ErrMismatch, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UPDATE
// ---------------------------------------------------------------------------

func TestUpdate_Basic(t *testing.T) {
	sql, args, err := pg.Update("users").
		Set("name", "Owen").
		Set("status", "active").
		Where("id", "=", 42).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `UPDATE "users" SET "name" = $1, "status" = $2 WHERE "id" = $3`)
	assertArgs(t, args, []any{"Owen", "active", 42})
}

func TestUpdate_SetRaw(t *testing.T) {
	sql, args, err := pg.Update("products").
		SetRaw("stock", "stock - 1").
		Where("id", "=", 7).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `UPDATE "products" SET "stock" = stock - 1 WHERE "id" = $1`)
	assertArgs(t, args, []any{7})
}

func TestUpdate_Returning(t *testing.T) {
	sql, _, err := pg.Update("users").
		Set("verified", true).
		Where("id", "=", 1).
		Returning("id", "verified").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `UPDATE "users" SET "verified" = $1 WHERE "id" = $2 RETURNING "id", "verified"`)
}

func TestUpdate_NoTable_Error(t *testing.T) {
	b := builder.NewUpdateBuilder(dialect.Postgres{})
	_, _, err := b.Set("a", 1).ToSQL()
	if !errors.Is(err, builder.ErrNoTable) {
		t.Errorf("expected ErrNoTable, got %v", err)
	}
}

func TestUpdate_NoColumns_Error(t *testing.T) {
	_, _, err := pg.Update("users").Where("id", "=", 1).ToSQL()
	if !errors.Is(err, builder.ErrNoColumns) {
		t.Errorf("expected ErrNoColumns, got %v", err)
	}
}

func TestUpdate_OrWhere(t *testing.T) {
	sql, args, err := pg.Update("users").
		Set("flagged", true).
		Where("role", "=", "bot").
		OrWhere("role", "=", "spam").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `UPDATE "users" SET "flagged" = $1 WHERE "role" = $2 OR "role" = $3`)
	assertArgs(t, args, []any{true, "bot", "spam"})
}

func TestUpdate_WhereGroup(t *testing.T) {
	sql, args, err := pg.Update("users").
		Set("archived", true).
		Where("active", "=", false).
		WhereGroup(func(b *builder.UpdateBuilder) {
			b.Where("plan", "=", "free").OrWhere("plan", "=", "trial")
		}).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `UPDATE "users" SET "archived" = $1 WHERE "active" = $2 AND ("plan" = $3 OR "plan" = $4)`)
	assertArgs(t, args, []any{true, false, "free", "trial"})
}

// ---------------------------------------------------------------------------
// DELETE
// ---------------------------------------------------------------------------

func TestDelete_Basic(t *testing.T) {
	sql, args, err := pg.Delete("users").Where("id", "=", 99).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `DELETE FROM "users" WHERE "id" = $1`)
	assertArgs(t, args, []any{99})
}

func TestDelete_Using(t *testing.T) {
	sql, args, err := pg.Delete("orders").
		Using("users").
		WhereRaw(`"orders"."user_id" = "users"."id"`).
		Where("users.status", "=", "banned").
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `DELETE FROM "orders" USING "users" WHERE "orders"."user_id" = "users"."id" AND "users"."status" = $1`)
	assertArgs(t, args, []any{"banned"})
}

func TestDelete_Returning(t *testing.T) {
	sql, _, err := pg.Delete("sessions").Where("user_id", "=", 1).Returning("id").ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `DELETE FROM "sessions" WHERE "user_id" = $1 RETURNING "id"`)
}

func TestDelete_NoTable_Error(t *testing.T) {
	b := builder.NewDeleteBuilder(dialect.Postgres{})
	_, _, err := b.ToSQL()
	if !errors.Is(err, builder.ErrNoTable) {
		t.Errorf("expected ErrNoTable, got %v", err)
	}
}

func TestDelete_OrWhere(t *testing.T) {
	sql, args, err := pg.Delete("sessions").
		Where("expired", "=", true).
		OrWhere("user_id", "=", 0).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `DELETE FROM "sessions" WHERE "expired" = $1 OR "user_id" = $2`)
	assertArgs(t, args, []any{true, 0})
}

func TestDelete_WhereGroup(t *testing.T) {
	sql, args, err := pg.Delete("logs").
		Where("level", "=", "debug").
		WhereGroup(func(b *builder.DeleteBuilder) {
			b.Where("service", "=", "worker").OrWhere("service", "=", "cron")
		}).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `DELETE FROM "logs" WHERE "level" = $1 AND ("service" = $2 OR "service" = $3)`)
	assertArgs(t, args, []any{"debug", "worker", "cron"})
}

// ---------------------------------------------------------------------------
// MySQL dialect
// ---------------------------------------------------------------------------

func TestMySQL_Placeholder(t *testing.T) {
	sql, args, err := my.Select("id", "email").
		From("users").
		Where("status", "=", "active").
		Where("age", ">", 18).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	// MySQL uses ? for all placeholders
	if strings.Count(sql, "?") != 2 {
		t.Errorf("expected 2 ? placeholders, got %s", sql)
	}
	assertArgs(t, args, []any{"active", 18})
	assertSQL(t, sql, "SELECT `id`, `email` FROM `users` WHERE `status` = ? AND `age` > ?")
}

func TestMySQL_QuoteIdentifier(t *testing.T) {
	d := dialect.MySQL{}
	got := d.QuoteIdentifier("user`name")
	want := "`user``name`"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Clone immutability
// ---------------------------------------------------------------------------

func TestSelect_WhereGroup_PropagatesInnerError(t *testing.T) {
	_, _, err := pg.Select("id").From("users").
		WhereGroup(func(b *builder.SelectBuilder) {
			b.Where("col", "INJECT", 1)
		}).
		ToSQL()
	if !errors.Is(err, builder.ErrInvalidOp) {
		t.Errorf("expected ErrInvalidOp from inner WhereGroup, got %v", err)
	}
}

func TestSelect_OrderBy_InvalidColumn_Error(t *testing.T) {
	_, _, err := pg.Select("id").From("users").OrderBy(`col"; SELECT`, builder.ASC).ToSQL()
	if !errors.Is(err, builder.ErrInvalidIdentifier) {
		t.Errorf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestSelect_GroupBy_InvalidColumn_Error(t *testing.T) {
	_, _, err := pg.Select("id").From("users").GroupBy(`bad; DROP`).ToSQL()
	if !errors.Is(err, builder.ErrInvalidIdentifier) {
		t.Errorf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestSelect_SafeExpression_Sleep_Rejected(t *testing.T) {
	_, _, err := pg.Select("SLEEP(5)").From("users").ToSQL()
	if !errors.Is(err, builder.ErrInvalidIdentifier) {
		t.Errorf("expected ErrInvalidIdentifier for SLEEP expression, got %v", err)
	}
}

func TestUpdate_WhereGroup_PropagatesInnerError(t *testing.T) {
	_, _, err := pg.Update("users").
		Set("x", 1).
		WhereGroup(func(b *builder.UpdateBuilder) {
			b.Where("col", "BAD", 1)
		}).
		ToSQL()
	if !errors.Is(err, builder.ErrInvalidOp) {
		t.Errorf("expected ErrInvalidOp, got %v", err)
	}
}

func TestSelect_MaliciousColumn_Rejected(t *testing.T) {
	_, _, err := pg.Select(`1; DROP TABLE users --`).From("users").ToSQL()
	if !errors.Is(err, builder.ErrInvalidIdentifier) {
		t.Errorf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestSelect_MaliciousSubqueryInColumn_Rejected(t *testing.T) {
	_, _, err := pg.Select("(SELECT secret FROM tokens LIMIT 1)").From("users").ToSQL()
	if !errors.Is(err, builder.ErrInvalidIdentifier) {
		t.Errorf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestSelect_Join_InvalidColumn_Error(t *testing.T) {
	_, _, err := pg.Select("id").From("users").
		Join("orders", "orders.user_id; DROP", "users.id").
		ToSQL()
	if !errors.Is(err, builder.ErrInvalidIdentifier) {
		t.Errorf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestUpdate_SetExpr(t *testing.T) {
	sql, args, err := pg.Update("products").
		SetExpr("stock", expr.Raw{SQL: "stock - 1"}).
		Where("id", "=", 7).
		ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `UPDATE "products" SET "stock" = stock - 1 WHERE "id" = $1`)
	assertArgs(t, args, []any{7})
}

func TestSelect_Clone_Immutable(t *testing.T) {
	base := pg.Select("id").From("users").Where("active", "=", true)
	clone := base.Clone()
	clone.Where("role", "=", "admin")

	basSQL, baseArgs, _ := base.ToSQL()
	cloneSQL, cloneArgs, _ := clone.ToSQL()

	if normalise(basSQL) == normalise(cloneSQL) {
		t.Error("clone mutation leaked into base builder")
	}
	if len(baseArgs) != 1 {
		t.Errorf("base should have 1 arg, got %d", len(baseArgs))
	}
	if len(cloneArgs) != 2 {
		t.Errorf("clone should have 2 args, got %d", len(cloneArgs))
	}
}
