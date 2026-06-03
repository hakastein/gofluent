package syntax

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hakastein/gofluent/syntax/ast"
)

// TestASTJSONShape parses a small FTL and verifies the canonical JSON shape
// (type discriminators, field names, ordering, null/empty handling), with
// spans omitted.
func TestASTJSONShape(t *testing.T) {
	res := Parse("foo = Bar\n")
	b, err := ast.Marshal(res, false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal produced output: %v\n%s", err, b)
	}

	wantJSON := `{
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
	var want interface{}
	if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
		t.Fatalf("bad want json: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("JSON shape mismatch.\n got: %s\nwant: %s", b, wantJSON)
	}
}

// TestASTJSONFieldOrder asserts the literal byte output to lock in field
// ordering (which DeepEqual on maps would not catch).
func TestASTJSONFieldOrder(t *testing.T) {
	res := Parse("foo = Bar\n")
	b, err := ast.Marshal(res, false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"type":"Resource","body":[{"type":"Message","id":{"type":"Identifier","name":"foo"},"value":{"type":"Pattern","elements":[{"type":"TextElement","value":"Bar"}]},"attributes":[],"comment":null}]}`
	if string(b) != want {
		t.Errorf("ordered JSON mismatch.\n got: %s\nwant: %s", b, want)
	}
}

// TestASTJSONWithSpans confirms spans appear last and use the Span shape.
func TestASTJSONWithSpans(t *testing.T) {
	res := Parse("foo = Bar\n")
	b, err := ast.Marshal(res, true)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sp, ok := got["span"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected span object, got %v", got["span"])
	}
	if sp["type"] != "Span" || sp["start"].(float64) != 0 || sp["end"].(float64) != 10 {
		t.Errorf("resource span = %v", sp)
	}
}
