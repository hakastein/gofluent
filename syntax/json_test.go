package syntax_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hakastein/gofluent/syntax"
	"github.com/hakastein/gofluent/syntax/ast"
)

// TestASTJSONShape parses a small FTL and verifies the canonical JSON shape
// (type discriminators, field names, null/empty handling), with spans omitted.
func TestASTJSONShape(t *testing.T) {
	res := syntax.Parse("foo = Bar\n")
	b, err := ast.Marshal(res, false)
	require.NoError(t, err)

	const wantJSON = `{
      "type": "Resource",
      "body": [
        {
          "type": "Message",
          "id": { "type": "Identifier", "name": "foo" },
          "value": {
            "type": "Pattern",
            "elements": [
              { "type": "TextElement", "value": "Bar" }
            ]
          },
          "attributes": [],
          "comment": null
        }
      ]
    }`
	assert.JSONEq(t, wantJSON, string(b))
}

// TestASTJSONFieldOrder asserts the literal byte output to lock in field
// ordering (which a structural JSON comparison would not catch).
func TestASTJSONFieldOrder(t *testing.T) {
	res := syntax.Parse("foo = Bar\n")
	b, err := ast.Marshal(res, false)
	require.NoError(t, err)

	const want = `{"type":"Resource","body":[{"type":"Message","id":{"type":"Identifier","name":"foo"},"value":{"type":"Pattern","elements":[{"type":"TextElement","value":"Bar"}]},"attributes":[],"comment":null}]}`
	assert.Equal(t, want, string(b))
}

// TestASTJSONWithSpans confirms spans appear and use the Span shape.
func TestASTJSONWithSpans(t *testing.T) {
	res := syntax.Parse("foo = Bar\n")
	b, err := ast.Marshal(res, true)
	require.NoError(t, err)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &got))

	sp, ok := got["span"].(map[string]interface{})
	require.True(t, ok, "expected span object, got %v", got["span"])
	assert.Equal(t, "Span", sp["type"])
	assert.Equal(t, float64(0), sp["start"])
	assert.Equal(t, float64(10), sp["end"])
}
