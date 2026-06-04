package syntax_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hakastein/gofluent/syntax"
	"github.com/hakastein/gofluent/syntax/ast"
)

// TestLineColumnOffsetUTF16 pins LineOffset/ColumnOffset to fluent.js semantics:
// pos is a UTF-16 code-unit offset (the parser emits UTF-16 offsets), not a Go
// byte offset. The reference lineOffset/columnOffset operate on JS strings,
// which are UTF-16. Sources with non-ASCII before pos must still resolve to the
// fluent.js-correct line/column.
func TestLineColumnOffsetUTF16(t *testing.T) {
	tests := []struct {
		name   string
		source string
		pos    int
		line   int
		column int
	}{
		{
			// Pure ASCII control: matches both byte and UTF-16 indexing.
			name:   "ascii",
			source: "ab\ncd",
			pos:    4, // 'd'
			line:   1,
			column: 1,
		},
		{
			// Each 'é' (U+00E9) is 1 UTF-16 unit but 2 UTF-8 bytes. pos is a
			// UTF-16 offset. Byte indexing would see only "éé" before pos 4 and
			// miss the newline.
			name:   "two-byte-bmp",
			source: "ééé\nx",
			pos:    4, // 'x'
			line:   1,
			column: 0,
		},
		{
			// Each '😀' (U+1F600) is 2 UTF-16 units (surrogate pair) but 4 bytes.
			name:   "astral-surrogate-pair",
			source: "😀😀\nXY",
			pos:    6, // 'Y'
			line:   1,
			column: 1,
		},
		{
			// columnOffset on the first line returns pos directly.
			name:   "first-line-bmp",
			source: "ééé\nx",
			pos:    2,
			line:   0,
			column: 2,
		},
		{
			// pos exactly at a newline: columnOffset uses fromIndex = pos-1 so the
			// break at pos is not counted as the previous one.
			name:   "pos-at-newline",
			source: "😀😀\nXY",
			pos:    4, // the '\n'
			line:   0,
			column: 4,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.line, syntax.LineOffset(tc.source, tc.pos), "LineOffset")
			assert.Equal(t, tc.column, syntax.ColumnOffset(tc.source, tc.pos), "ColumnOffset")
		})
	}
}

// TestLineColumnOffsetFromJunkSpan exercises the end-to-end path the helpers
// exist for: a Junk span start is a UTF-16 unit offset, and feeding it to the
// helpers must locate the junk correctly even with multibyte text before it.
func TestLineColumnOffsetFromJunkSpan(t *testing.T) {
	// Line 0 is a valid message with a 2-byte char; line 1 is junk.
	src := "ok = é\n123\n"
	res := syntax.Parse(src)

	var junk *ast.Junk
	for _, e := range res.Body {
		if j, ok := e.(*ast.Junk); ok {
			junk = j
			break
		}
	}
	require.NotNil(t, junk, "expected a Junk entry")
	sp := junk.GetSpan()
	require.NotNil(t, sp, "expected junk to carry a span")

	// The junk begins at the start of the second line.
	assert.Equal(t, 1, syntax.LineOffset(src, sp.Start), "junk should be on line index 1")
	assert.Equal(t, 0, syntax.ColumnOffset(src, sp.Start), "junk should start at column 0")
}

// countingVisitor records every node type it sees.
type countingVisitor struct {
	seen map[string]int
}

func (c *countingVisitor) Visit(node ast.Node) bool {
	if c.seen == nil {
		c.seen = map[string]int{}
	}
	switch node.(type) {
	case *ast.Span:
		c.seen["Span"]++
	case *ast.Annotation:
		c.seen["Annotation"]++
	}
	return true
}

// TestWalkVisitsSpans verifies Walk descends into Span nodes, matching the
// reference genericVisit which recurses into the `span` property.
func TestWalkVisitsSpans(t *testing.T) {
	res := syntax.Parse("msg = hi { $x }\n")
	v := &countingVisitor{}
	syntax.Walk(v, res)
	assert.Greater(t, v.seen["Span"], 0, "Walk must visit at least one *ast.Span")
}

// identityTransformer returns every node unchanged, also recording whether it
// was handed any Span node.
type identityTransformer struct {
	sawSpan bool
}

func (it *identityTransformer) Transform(node ast.Node) ast.Node {
	if _, ok := node.(*ast.Span); ok {
		it.sawSpan = true
	}
	return node
}

// spanStrippingTransformer drops every Span node it is handed (returns nil),
// leaving all other nodes intact.
type spanStrippingTransformer struct{}

func (spanStrippingTransformer) Transform(node ast.Node) ast.Node {
	if _, ok := node.(*ast.Span); ok {
		return nil
	}
	return node
}

func TestTransformVisitsSpans(t *testing.T) {
	src := "msg = hi { $x }\n"

	// A no-op transformer must preserve spans and round-trip identically.
	res := syntax.Parse(src)
	it := &identityTransformer{}
	out := syntax.Transform(it, res)
	assert.True(t, it.sawSpan, "Transform must hand the transformer at least one *ast.Span")
	assert.Equal(t, src, syntax.Serialize(out.(*ast.Resource)), "identity Transform must round-trip")

	// A span-stripping transformer must remove every span yet keep the tree
	// serializable.
	res2 := syntax.Parse(src)
	out2 := syntax.Transform(spanStrippingTransformer{}, res2)
	r2 := out2.(*ast.Resource)
	assert.Nil(t, r2.GetSpan(), "resource span should be stripped")
	require.NotEmpty(t, r2.Body)
	assert.Nil(t, r2.Body[0].GetSpan(), "message span should be stripped")
	assert.Equal(t, src, syntax.Serialize(r2), "span-stripped tree must still serialize")
}
