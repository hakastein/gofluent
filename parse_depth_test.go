package fluent_test

import (
	"strings"
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuntimeDeepNestingSkipped verifies the runtime parser stays fault-tolerant
// under pathologically nested placeables: the bad entry is skipped (as any
// malformed entry is) rather than crashing the process with a fatal, unrecover-
// able Go stack overflow.
func TestRuntimeDeepNestingSkipped(t *testing.T) {
	const depth = 10_000
	src := "foo = " + strings.Repeat("{ ", depth) + "$x" + strings.Repeat(" }", depth) + "\n" +
		"good = Value\n"

	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
	require.NotPanics(t, func() { b.AddResource(fluent.NewResource(src)) })

	_, ok := b.Message("foo")
	assert.False(t, ok, "deeply nested entry should be skipped")

	_, ok = b.Message("good")
	assert.True(t, ok, "the following entry should still parse")
}

// TestRuntimeModestNestingParses confirms the depth cap leaves reasonable
// nesting intact and resolving correct.
func TestRuntimeModestNestingParses(t *testing.T) {
	const depth = 10
	src := "foo = " + strings.Repeat("{ ", depth) + "$x" + strings.Repeat(" }", depth) + "\n"

	b := newTestBundle(t, src)
	got, err := format(t, b, "foo", map[string]any{"x": "hi"})
	assert.Equal(t, "hi", got)
	assert.NoError(t, err)
}
