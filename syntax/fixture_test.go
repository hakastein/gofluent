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

// fixturesDir points at the vendored structure fixtures under
// internal/conformance/testdata/structure. These files are tracked in the
// repository and must be present on every checkout; a missing file is a test
// failure, not a skip. The full fixture sweep is owned by the conformance
// suite; this file holds a focused spot-check on a few representative cases.
const fixturesDir = "../internal/conformance/testdata/structure"

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
			ftlPath := filepath.Join(fixturesDir, name+".ftl")
			jsonPath := filepath.Join(fixturesDir, name+".json")

			ftl, err := os.ReadFile(ftlPath)
			require.NoError(t, err, "reading fixture %s", ftlPath)
			wantBytes, err := os.ReadFile(jsonPath)
			require.NoError(t, err, "reading expected JSON %s", jsonPath)

			res := syntax.Parse(string(ftl))
			gotBytes, err := ast.Marshal(res, true)
			require.NoError(t, err, "marshal")

			assert.JSONEq(t, string(wantBytes), string(gotBytes), "AST mismatch for %s", name)
		})
	}
}
