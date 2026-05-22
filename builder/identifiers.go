package builder

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/Software78/sql-go-query-builder/internal/dialect"
)

var forbiddenSQLWords = regexp.MustCompile(
	`(?i)\b(select|from|union|insert|delete|update|drop|exec|execute|sleep|benchmark|pg_sleep|waitfor|delay)\b`,
)

// quoteQualifiedCol quotes each dot-separated segment as an identifier.
func quoteQualifiedCol(d dialect.Dialect, col string) string {
	parts := strings.Split(col, ".")
	for i, p := range parts {
		parts[i] = d.QuoteIdentifier(p)
	}
	return strings.Join(parts, ".")
}

// joinCondition builds a safe ON clause from two qualified column references.
func joinCondition(d dialect.Dialect, leftCol, rightCol string) string {
	return quoteQualifiedCol(d, leftCol) + " = " + quoteQualifiedCol(d, rightCol)
}

// validateIdentifierColumn reports whether a name is a simple or qualified identifier.
func validateIdentifierColumn(col string) error {
	if isSimpleIdentifier(col) || isQualifiedColumn(col) {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidIdentifier, col)
}

// quoteIdentifierColumn quotes a simple or qualified column reference.
func quoteIdentifierColumn(d dialect.Dialect, col string) string {
	if isQualifiedColumn(col) {
		return quoteQualifiedCol(d, col)
	}
	return d.QuoteIdentifier(col)
}

// validateSelectColumn reports whether a Select() argument is permitted.
func validateSelectColumn(c string) error {
	if c == "*" || isQualifiedColumn(c) || isSafeSelectExpression(c) || isSimpleIdentifier(c) {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidIdentifier, c)
}

// quoteSelectCol renders a SELECT list entry with injection-safe rules.
func quoteSelectCol(d dialect.Dialect, c string) string {
	switch {
	case c == "*":
		return c
	case isQualifiedColumn(c):
		return quoteQualifiedCol(d, c)
	case isSafeSelectExpression(c):
		return c
	default:
		return d.QuoteIdentifier(c)
	}
}

func isSimpleIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
		} else if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isQualifiedColumn(c string) bool {
	if !strings.Contains(c, ".") {
		return false
	}
	if strings.ContainsAny(c, " ();\"'\\\x00\n\r\t,;") {
		return false
	}
	for _, part := range strings.Split(c, ".") {
		if !isSimpleIdentifier(part) {
			return false
		}
	}
	return true
}

func isValidJoinColumn(c string) bool {
	if c == "" {
		return false
	}
	if strings.ContainsAny(c, " ();\"'\\\x00\n\r\t,;") {
		return false
	}
	for _, part := range strings.Split(c, ".") {
		if !isSimpleIdentifier(part) {
			return false
		}
	}
	return true
}

// isSafeSelectExpression allows function/aggregate expressions that cannot be
// expressed as a simple quoted column (e.g. COUNT(*) AS cnt). Rejects obvious
// injection patterns; prefer expr.Expr for complex projections.
func isSafeSelectExpression(c string) bool {
	lower := strings.ToLower(c)
	for _, forbidden := range []string{";", "--", "/*", "*/", "xp_", "0x"} {
		if strings.Contains(lower, forbidden) {
			return false
		}
	}
	if forbiddenSQLWords.MatchString(c) {
		return false
	}
	if !strings.Contains(c, "(") {
		return false
	}
	for _, r := range c {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
		case r == '_', r == '.', r == '*', r == '(', r == ')', r == ' ', r == ',':
		default:
			return false
		}
	}
	return true
}
