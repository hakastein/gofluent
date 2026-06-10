package syntax_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hakastein/gofluent/syntax"
)

// Round-trip and idempotency over real-world inputs are covered by the
// conformance suite (internal/conformance); these tests cover only the
// serializer options the suite does not exercise.

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
