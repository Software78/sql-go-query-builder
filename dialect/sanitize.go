package dialect

import "strings"

// sanitizeIdentifier strips null bytes and ASCII control characters that must
// not appear in SQL identifiers.
func sanitizeIdentifier(s string) string {
	return strings.Map(func(r rune) rune {
		if r == 0 || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, s)
}
