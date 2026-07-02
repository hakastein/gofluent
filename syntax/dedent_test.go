package syntax_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hakastein/gofluent/syntax"
	"github.com/hakastein/gofluent/syntax/ast"
)

// patternValue returns the single message's pattern elements.
func patternElements(t *testing.T, src string) []ast.PatternElement {
	t.Helper()
	res := syntax.Parse(src)
	require.Len(t, res.Body, 1)
	msg, ok := res.Body[0].(*ast.Message)
	require.Truef(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.NotNil(t, msg.Value)
	return msg.Value.Elements
}

// TestDedentMergesTextRuns locks the block-pattern dedent contract: adjacent
// text lines collapse into a single TextElement, placeables break a run into
// separate TextElements, trailing blank lines are trimmed, and CRLF is
// normalized to LF. This is the correctness guard for the O(n) merge rewrite.
func TestDedentMergesTextRuns(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []ast.PatternElement
	}{
		{
			name: "multiline block merges into one text element",
			src:  "foo =\n    Line one\n    Line two\n    Line three\n",
			want: []ast.PatternElement{
				&ast.TextElement{Value: "Line one\nLine two\nLine three"},
			},
		},
		{
			name: "trailing blank lines are trimmed",
			src:  "foo =\n    Text\n    \n\n",
			want: []ast.PatternElement{
				&ast.TextElement{Value: "Text"},
			},
		},
		{
			name: "inline first line merges with continuation",
			src:  "foo = A\n    B\n",
			want: []ast.PatternElement{
				&ast.TextElement{Value: "A\nB"},
			},
		},
		{
			name: "placeable splits text into separate runs",
			src:  "foo =\n    A { $x } B\n    C\n",
			want: []ast.PatternElement{
				&ast.TextElement{Value: "A "},
				nil, // placeholder for the placeable, checked separately
				&ast.TextElement{Value: " B\nC"},
			},
		},
		{
			name: "crlf is normalized to lf when merging",
			src:  "foo =\r\n    Line one\r\n    Line two\r\n",
			want: []ast.PatternElement{
				&ast.TextElement{Value: "Line one\nLine two"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := patternElements(t, tc.src)
			require.Len(t, got, len(tc.want))
			for i, want := range tc.want {
				if want == nil {
					_, ok := got[i].(*ast.Placeable)
					assert.Truef(t, ok, "element %d: expected *ast.Placeable, got %T", i, got[i])
					continue
				}
				te, ok := got[i].(*ast.TextElement)
				require.Truef(t, ok, "element %d: expected *ast.TextElement, got %T", i, got[i])
				assert.Equal(t, want.(*ast.TextElement).Value, te.Value)
			}
		})
	}
}

// TestDedentLargeBlockIsLinear is a sanity bound: a large block pattern must
// parse quickly. The previous merge was O(n^2) and took tens of seconds at this
// size; the linear merge finishes in well under a second. It is a coarse guard
// against a quadratic regression, not a precise benchmark.
func TestDedentLargeBlockIsLinear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-input timing guard in -short mode")
	}

	const lines = 200_000
	var b strings.Builder
	b.WriteString("foo =\n")
	for i := 0; i < lines; i++ {
		b.WriteString("    x\n")
	}

	res := syntax.Parse(b.String())
	require.Len(t, res.Body, 1)
	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok)
	require.NotNil(t, msg.Value)
	require.Len(t, msg.Value.Elements, 1)
}
