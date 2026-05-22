# qb — Go SQL Query Builder

A production-grade, database-agnostic SQL query builder for Go.
No ORM. No reflection. No external dependencies. Just composable, safe SQL.

## Features

- **SELECT** with JOIN, WHERE, GROUP BY, HAVING, ORDER BY, LIMIT, OFFSET, FOR UPDATE, CTEs, subqueries
- **INSERT** with bulk values, `INSERT INTO ... SELECT`, `ON CONFLICT DO NOTHING / DO UPDATE`, `RETURNING`
- **UPDATE** with raw expressions (`SET stock = stock - 1`), `RETURNING`
- **DELETE** with `USING` (PostgreSQL), `RETURNING`
- Recursive **predicate tree** — `WhereGroup` produces correctly parenthesised nested conditions
- Globally correct **placeholder indices** (`$1`, `$2`, ...) across composed subqueries
- **Dialect abstraction** — PostgreSQL (`$N`, `"ident"`) and MySQL (`?`, `` `ident` ``) built in
- **Clone** — branch a shared base builder without mutation
- Typed **sentinel errors** — `ErrNoTable`, `ErrMismatch`, etc.
- Zero dependencies — stdlib only

## Quick start

```go
q := qb.New(dialect.Postgres{})
```

## SELECT

```go
sql, args, err := q.Select("id", "email", "status").
    From("users").
    Where("status", "=", "active").
    Where("age", ">", 18).
    OrderBy("created_at", builder.DESC).
    Limit(20).
    Offset(0).
    ToSQL()

// SELECT "id", "email", "status"
// FROM "users"
// WHERE "status" = $1 AND "age" > $2
// ORDER BY "created_at" DESC
// LIMIT 20 OFFSET 0
//
// args: ["active", 18]
```

### WHERE variants

```go
// IN / NOT IN
q.Select("id").From("orders").WhereIn("status", "paid", "shipped")
q.Select("id").From("orders").WhereNotIn("state", "cancelled", "refunded")

// NULL checks
q.Select("id").From("users").WhereNull("deleted_at")
q.Select("id").From("users").WhereNotNull("email")

// BETWEEN
q.Select("id").From("orders").WhereBetween("amount", 100, 500)

// Raw predicate (caller is responsible for safety)
q.Select("id").From("users").WhereRaw("LOWER(email) = ?", "test@example.com")
// Use ?? for a literal ? (e.g. PostgreSQL JSONB): WhereRaw("metadata ?? ?", key)

// Grouped conditions — produces (role = $1 OR role = $2)
q.Select("id").From("users").
    Where("active", "=", true).
    WhereGroup(func(b *builder.SelectBuilder) {
        b.Where("role", "=", "admin").OrWhere("role", "=", "owner")
    })
```

### JOINs

```go
q.Select("u.id", "o.total").
    From("users").
    Join("orders", "orders.user_id", "users.id").
    LeftJoin("profiles", "profiles.user_id", "users.id")
    // JoinRaw(table, condition) for complex ON clauses — caller-sanitised only
```

### Subquery in FROM

```go
sub := q.Select("id", "email").From("users").Where("active", "=", true)

q.Select("id").FromSubquery(sub, "active_users").ToSQL()
// SELECT "id" FROM (SELECT "id", "email" FROM "users" WHERE "active" = $1) "active_users"
```

### Common Table Expressions (CTEs)

```go
cte := q.Select("id", "amount").From("orders").Where("status", "=", "paid")

q.Select("id").
    From("paid_orders").
    WithCTE("paid_orders", cte).
    ToSQL()
// WITH "paid_orders" AS (SELECT ...) SELECT "id" FROM "paid_orders"
```

### Clone — branching a base query

```go
base := q.Select("id").From("users").Where("active", "=", true)

adminQ := base.Clone().Where("role", "=", "admin")
ownerQ := base.Clone().Where("role", "=", "owner")
// base is unchanged
```

## INSERT

```go
// Single row
q.Insert("users").
    Columns("name", "email").
    Values("Owen", "owen@example.com").
    ToSQL()

// Bulk insert
q.Insert("users").
    Columns("name", "email").
    BulkValues([][]any{
        {"Alice", "alice@example.com"},
        {"Bob",   "bob@example.com"},
    }).
    ToSQL()

// Upsert — ON CONFLICT DO UPDATE
q.Insert("users").
    Columns("email", "name").
    Values("owen@example.com", "Owen").
    OnConflict("email").DoUpdate("name", "Owen Updated").Back().
    Returning("id", "updated_at").
    ToSQL()

// INSERT INTO ... SELECT
sel := q.Select("name", "email").From("staging").Where("verified", "=", true)
q.Insert("users").Columns("name", "email").FromSelect(sel).ToSQL()
```

## UPDATE

```go
q.Update("users").
    Set("name", "Owen").
    Set("status", "active").
    Where("id", "=", 42).
    Returning("id", "updated_at").
    ToSQL()

// Raw expression
q.Update("products").
    SetExpr("stock", expr.Raw{SQL: "stock - 1"}).
    Where("id", "=", 7).
    ToSQL()

// Map of updates
q.Update("users").
    SetMap(map[string]any{"verified": true, "role": "admin"}).
    Where("id", "=", 1).
    ToSQL()
```

## DELETE

```go
q.Delete("users").Where("id", "=", 99).ToSQL()

// PostgreSQL DELETE ... USING
q.Delete("orders").
    Using("users").
    WhereRaw(`"orders"."user_id" = "users"."id"`).
    Where("users.status", "=", "banned").
    Returning("id").
    ToSQL()
```

## MySQL dialect

```go
myq := qb.New(dialect.MySQL{})

myq.Select("id").From("users").Where("active", "=", true).ToSQL()
// SELECT `id` FROM `users` WHERE `active` = ?
```

## Error handling

All `ToSQL()` calls return `(string, []any, error)` — they never panic.

```go
sql, args, err := q.Select("id").ToSQL() // forgot .From(...)
if errors.Is(err, builder.ErrNoTable) {
    // handle
}
```

Sentinel errors: `ErrNoTable`, `ErrNoColumns`, `ErrNoValues`, `ErrMismatch`,
`ErrInvalidOp`, `ErrEmptyIN`, `ErrNilSubquery`, `ErrNoAlias`.

## Thread safety

Builders are **not goroutine-safe** by design. Use `Clone()` to branch a shared
base builder if you need to construct related queries concurrently.

## Adding a dialect

Implement the `dialect.Dialect` interface:

```go
type Dialect interface {
    Placeholder(n int) string
    QuoteIdentifier(s string) string
    LimitClause(limit, offset int64) string
    Name() string
}
```
