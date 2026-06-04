package syntax_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hakastein/gofluent/syntax"
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

			res := syntax.Parse(string(ftl))
			gotBytes, err := ast.Marshal(res, true)
			require.NoError(t, err, "marshal")

			assert.JSONEq(t, string(wantBytes), string(gotBytes), "AST mismatch for %s", name)
		})
	}
}
