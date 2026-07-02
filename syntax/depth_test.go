package syntax_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hakastein/gofluent/syntax"
	"github.com/hakastein/gofluent/syntax/ast"
)

// TestParseDeepNestingBecomesJunk verifies that pathologically nested placeables
// do not crash the process (a Go stack overflow is fatal and unrecoverable) but
// instead fail through the normal Junk-with-annotation path.
func TestParseDeepNestingBecomesJunk(t *testing.T) {
	const depth = 10_000
	src := "foo = " + strings.Repeat("{", depth) + "$x" + strings.Repeat("}", depth) + "\n"

	var res *ast.Resource
	require.NotPanics(t, func() { res = syntax.Parse(src) })

	var codes []string
	for _, entry := range res.Body {
		if junk, ok := entry.(*ast.Junk); ok {
			for _, a := range junk.Annotations {
				codes = append(codes, a.Code)
			}
		}
	}
	assert.NotEmpty(t, codes, "deeply nested placeables should yield Junk with an annotation")
}

// TestParseModestNestingParses confirms the depth cap does not reject nesting
// that real translations might plausibly use.
func TestParseModestNestingParses(t *testing.T) {
	const depth = 10
	src := "foo = " + strings.Repeat("{", depth) + "$x" + strings.Repeat("}", depth) + "\n"

	res := syntax.Parse(src)
	require.Len(t, res.Body, 1)
	msg, ok := res.Body[0].(*ast.Message)
	require.Truef(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.NotNil(t, msg.Value)
}
