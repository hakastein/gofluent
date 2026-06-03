package plural

import (
	"encoding/json"
	"os"
	"testing"
)

// sampleRow is one expanded CLDR sample: the value string (preserving its
// fraction-digit width) and the plural category CLDR declares for it.
type sampleRow struct {
	Type     string `json:"type"`
	Locale   string `json:"locale"`
	Category string `json:"category"`
	Value    string `json:"value"`
}

// TestCLDRSamples is a self-consistency proof straight from CLDR: every
// @integer / @decimal sample listed under a category must be classified into
// that same category by the generated rules, for every locale in the data.
//
// The samples are committed under testdata/cldr_samples.json, produced by
// internal/gen/samples.js (see that file to regenerate).
func TestCLDRSamples(t *testing.T) {
	data, err := os.ReadFile("testdata/cldr_samples.json")
	if err != nil {
		t.Fatalf("read samples: %v", err)
	}
	var rows []sampleRow
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("unmarshal samples: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("no samples loaded")
	}

	var fails int
	for _, r := range rows {
		ops, err := OperandsFromString(r.Value)
		if err != nil {
			t.Errorf("%s %s %q: OperandsFromString: %v", r.Type, r.Locale, r.Value, err)
			fails++
			continue
		}
		var got Category
		switch r.Type {
		case "cardinal":
			got = Cardinal(r.Locale, ops)
		case "ordinal":
			got = Ordinal(r.Locale, ops)
		default:
			t.Fatalf("unknown type %q", r.Type)
		}
		if string(got) != r.Category {
			fails++
			if fails <= 50 {
				t.Errorf("%s %s value=%q: got %q want %q (ops=%+v)",
					r.Type, r.Locale, r.Value, got, r.Category, ops)
			}
		}
	}
	if fails > 0 {
		t.Errorf("CLDR sample mismatches: %d / %d", fails, len(rows))
	} else {
		t.Logf("CLDR self-test: %d/%d samples matched", len(rows), len(rows))
	}
}
