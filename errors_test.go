package fluent_test

import (
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ported from errors_test.js and bomb_test.js.

func TestErrorReportingIntoArray(t *testing.T) {
	b := newTestBundle(t, "foo = {$one} and {$two}\n")
	msg, _ := b.GetMessage("foo")

	var errs []error
	val := b.FormatPattern(msg.Value, map[string]fluent.Value{}, &errs)
	assert.Equal(t, "{$one} and {$two}", val)
	require.Len(t, errs, 2, "expected 2 reference errors")
	require.ErrorIs(t, errs[0], fluent.ErrReference)
	require.ErrorIs(t, errs[1], fluent.ErrReference)

	val = b.FormatPattern(msg.Value, map[string]fluent.Value{}, &errs)
	assert.Equal(t, "{$one} and {$two}", val)
	require.Len(t, errs, 4, "errors accumulate across calls into the same sink")
	require.ErrorIs(t, errs[2], fluent.ErrReference)
	require.ErrorIs(t, errs[3], fluent.ErrReference)
}

func TestFirstErrorIsThrown(t *testing.T) {
	b := newTestBundle(t, "foo = {$one} and {$two}\n")
	msg, _ := b.GetMessage("foo")

	// With a nil error sink the first error is "thrown" (panics).
	assert.Panics(t, func() {
		b.FormatPattern(msg.Value, nil, nil)
	}, "expected panic when errs is nil")
}

func TestBillionLaughs(t *testing.T) {
	src := "lol0 = LOL\n" +
		"lol1 = {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0}\n" +
		"lol2 = {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1}\n" +
		"lol3 = {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2}\n" +
		"lol4 = {lol3} {lol3} {lol3} {lol3} {lol3} {lol3} {lol3} {lol3} {lol3} {lol3}\n" +
		"lol5 = {lol4} {lol4} {lol4} {lol4} {lol4} {lol4} {lol4} {lol4} {lol4} {lol4}\n" +
		"lol6 = {lol5} {lol5} {lol5} {lol5} {lol5} {lol5} {lol5} {lol5} {lol5} {lol5}\n" +
		"lol7 = {lol6} {lol6} {lol6} {lol6} {lol6} {lol6} {lol6} {lol6} {lol6} {lol6}\n" +
		"lol8 = {lol7} {lol7} {lol7} {lol7} {lol7} {lol7} {lol7} {lol7} {lol7} {lol7}\n" +
		"lol9 = {lol8} {lol8} {lol8} {lol8} {lol8} {lol8} {lol8} {lol8} {lol8} {lol8}\n" +
		"lolz = {lol9}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "lolz", nil)
	assert.Equal(t, "{???}", got)
	require.Len(t, errs, 1, "expected a single range error")
	require.ErrorIs(t, errs[0], fluent.ErrRange)
}

func TestBillionLaughsThrowsWithoutSink(t *testing.T) {
	src := "lol0 = LOL\n" +
		"lol1 = {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0}\n" +
		"lol2 = {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1}\n" +
		"lol3 = {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2}\n" +
		"lolz = {lol3}\n"
	b := newTestBundle(t, src)
	msg, ok := b.GetMessage("lolz")
	require.True(t, ok)

	// With a nil error sink the range error is "thrown" (panics).
	assert.Panics(t, func() {
		b.FormatPattern(msg.Value, nil, nil)
	}, "expected panic")
}
