package plural

import (
	"encoding/json"
	"os"
	"testing"
)

// intlRow is one Intl.PluralRules result captured from Node/V8 (full-ICU).
type intlRow struct {
	Locale   string `json:"locale"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	MinFrac  int    `json:"minFrac"`
	MaxFrac  int    `json:"maxFrac"`
	Category string `json:"category"`
}

// TestIntlParity asserts the generated tables agree with JavaScript's
// Intl.PluralRules over a wide locale x type x value matrix. Both derive from
// the same CLDR data, so the match rate must be 100%; any mismatch signals an
// operand or parsing bug.
//
// The matrix is committed under testdata/intl_plurals.json, produced by
// internal/gen/intl.js (see that file to regenerate).
func TestIntlParity(t *testing.T) {
	data, err := os.ReadFile("testdata/intl_plurals.json")
	if err != nil {
		t.Fatalf("read intl data: %v", err)
	}
	var rows []intlRow
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("unmarshal intl data: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("no intl rows loaded")
	}

	type stat struct{ ok, total int }
	perLocale := map[string]*stat{}
	var fails, total int

	for _, r := range rows {
		// Use the string form (authoritative for v/w/f/t), matching the
		// fraction-digit shape Intl was given.
		ops, err := OperandsFromString(r.Value)
		if err != nil {
			t.Fatalf("OperandsFromString(%q): %v", r.Value, err)
		}
		var got Category
		switch r.Type {
		case "cardinal":
			got = Cardinal(r.Locale, ops)
		case "ordinal":
			got = Ordinal(r.Locale, ops)
		}
		key := r.Locale + "/" + r.Type
		s := perLocale[key]
		if s == nil {
			s = &stat{}
			perLocale[key] = s
		}
		s.total++
		total++
		if string(got) == r.Category {
			s.ok++
		} else {
			fails++
			if fails <= 50 {
				t.Errorf("%s %s value=%q: got %q want %q (ops=%+v)",
					r.Type, r.Locale, r.Value, got, r.Category, ops)
			}
		}
	}

	matched := total - fails
	t.Logf("Intl parity overall: %d/%d (%.4f%%)", matched, total,
		100*float64(matched)/float64(total))
	if fails != 0 {
		// Report worst locales to aid debugging.
		for key, s := range perLocale {
			if s.ok != s.total {
				t.Logf("  mismatch in %s: %d/%d", key, s.ok, s.total)
			}
		}
		t.Errorf("Intl parity mismatches: %d / %d", fails, total)
	}
}

// TestNewOperandsParity checks the float-based NewOperands path against Intl as
// well, ensuring the fraction-digit formatting logic produces the same
// operands as the string path for the same min/max fraction digits.
func TestNewOperandsParity(t *testing.T) {
	data, err := os.ReadFile("testdata/intl_plurals.json")
	if err != nil {
		t.Fatalf("read intl data: %v", err)
	}
	var rows []intlRow
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("unmarshal intl data: %v", err)
	}

	var fails int
	for _, r := range rows {
		var got Category
		switch r.Type {
		case "cardinal":
			got = CardinalFor(r.Locale, parseF(r.Value), r.MinFrac, r.MaxFrac)
		case "ordinal":
			got = OrdinalFor(r.Locale, parseF(r.Value), r.MinFrac, r.MaxFrac)
		}
		if string(got) != r.Category {
			fails++
			if fails <= 30 {
				t.Errorf("%s %s value=%q minF=%d maxF=%d: got %q want %q",
					r.Type, r.Locale, r.Value, r.MinFrac, r.MaxFrac, got, r.Category)
			}
		}
	}
	if fails != 0 {
		t.Errorf("NewOperands parity mismatches: %d / %d", fails, len(rows))
	}
}

func parseF(s string) float64 {
	ops, err := OperandsFromString(s)
	if err != nil {
		return 0
	}
	return ops.N
}
