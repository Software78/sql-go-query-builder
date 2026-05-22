package dialect

import "testing"

func TestQuoteIdentifier_StripsControlChars(t *testing.T) {
	pg := Postgres{}
	got := pg.QuoteIdentifier("users\nWHERE 1=1")
	want := `"usersWHERE 1=1"`
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}

	my := MySQL{}
	got = my.QuoteIdentifier("col\x00name")
	want = "`colname`"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
