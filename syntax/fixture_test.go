package syntax

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hakastein/gofluent/syntax/ast"
)

// fixturesDir points at the reference fixtures_structure directory. These are
// only used for a few spot checks here; the full sweep is owned by the
// conformance suite.
const fixturesDir = "../.reference/fluent.js/fluent-syntax/test/fixtures_structure"

func TestFixtureSpotChecks(t *testing.T) {
	names := []string{
		"junk",
		"leading_dots", // contains a non-ASCII char (…) -> exercises UTF-16 spans
		"escape_sequences",
		"crlf",
		"variant_with_empty_pattern",
		"expressions_call_args",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ftl, err := os.ReadFile(filepath.Join(fixturesDir, name+".ftl"))
			if err != nil {
				t.Skipf("fixture missing: %v", err)
			}
			wantBytes, err := os.ReadFile(filepath.Join(fixturesDir, name+".json"))
			if err != nil {
				t.Skipf("expected json missing: %v", err)
			}

			res := Parse(string(ftl))
			gotBytes, err := ast.Marshal(res, true)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var got, want interface{}
			if err := json.Unmarshal(gotBytes, &got); err != nil {
				t.Fatalf("unmarshal got: %v", err)
			}
			if err := json.Unmarshal(wantBytes, &want); err != nil {
				t.Fatalf("unmarshal want: %v", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("AST mismatch for %s.\n got: %s", name, gotBytes)
			}
		})
	}
}
