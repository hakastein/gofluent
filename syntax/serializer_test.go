package syntax_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hakastein/gofluent/syntax"
)

func TestSerializeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "simple message", src: "foo = Bar\n"},
		{name: "term", src: "-brand = Firefox\n"},
		{name: "message with attribute", src: "foo = Bar\n    .attr = Baz\n"},
		{name: "multiline pattern", src: "foo =\n    Multi\n    line\n"},
		{name: "select expression", src: "foo = { $n ->\n        [one] One\n       *[other] Other\n    }\n"},
		{name: "call arguments", src: "foo = { FUN($a, key: 1) }\n"},
		{name: "placeable in text", src: "foo = Pre { $var } post\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := syntax.Serialize(syntax.Parse(tc.src))
			// Re-parse and re-serialize to confirm a stable canonical form.
			got2 := syntax.Serialize(syntax.Parse(got))
			assert.Equal(t, got, got2, "serialization not idempotent")
		})
	}
}

func TestSerializeStableForCanonicalInput(t *testing.T) {
	// These inputs are already in canonical form; serialize must reproduce them.
	tests := []struct {
		name string
		src  string
	}{
		{name: "simple message", src: "foo = Bar\n"},
		{name: "term", src: "-brand = Firefox\n"},
		{name: "message with attribute", src: "foo = Bar\n    .attr = Baz\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := syntax.Serialize(syntax.Parse(tc.src))
			assert.Equal(t, tc.src, got, "serialize must be identity for canonical input")
		})
	}
}

func TestSerializeWithJunk(t *testing.T) {
	src := "err = {1xx}\ngood = Value\n"
	res := syntax.Parse(src)

	withoutJunk := syntax.Serialize(res)
	assert.Equal(t, "good = Value\n", withoutJunk, "junk omitted by default")

	withJunk := syntax.Serialize(res, syntax.WithJunk(true))
	assert.Equal(t, src, withJunk, "junk preserved with WithJunk(true)")
}

func TestSerializeComment(t *testing.T) {
	src := "# A comment\nfoo = Bar\n"
	got := syntax.Serialize(syntax.Parse(src))
	assert.Equal(t, src, got)
}
