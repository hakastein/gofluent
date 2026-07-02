package fluent_test

import (
	"fmt"
	"strings"
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ported from errors_test.js and bomb_test.js.

func TestErrorsAreCollected(t *testing.T) {
	b := newTestBundle(t, "foo = {$one} and {$two}\n")

	got, err := format(t, b, "foo", nil)
	assert.Equal(t, "{$one} and {$two}", got, "a best-effort string is returned despite errors")
	errs := errList(err)
	require.Len(t, errs, 2, "expected 2 reference errors")
	require.ErrorIs(t, errs[0], fluent.ErrReference)
	require.ErrorIs(t, errs[1], fluent.ErrReference)
}

// lolSource builds a billion-laughs chain of the given depth: lolN expands to
// ten copies of lolN-1.
func lolSource(depth int) string {
	var sb strings.Builder
	sb.WriteString("lol0 = LOL\n")
	for i := 1; i <= depth; i++ {
		refs := strings.TrimSuffix(strings.Repeat(fmt.Sprintf("{lol%d} ", i-1), 10), " ")
		fmt.Fprintf(&sb, "lol%d = %s\n", i, refs)
	}
	fmt.Fprintf(&sb, "lolz = {lol%d}\n", depth)
	return sb.String()
}

func TestBillionLaughs(t *testing.T) {
	b := newTestBundle(t, lolSource(9))

	got, err := format(t, b, "lolz", nil)
	assert.Equal(t, "{???}", got, "the placeable limit aborts expansion with a fallback")
	require.Len(t, errList(err), 1, "expected a single range error")
	require.ErrorIs(t, err, fluent.ErrRange)
}
