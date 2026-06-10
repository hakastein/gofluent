package fluent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Ported from primitives_test.js.

// TestUnicodeEscapeOutOfRange covers a \U escape whose codepoint exceeds the
// Unicode maximum (U+10FFFF). It must render as U+FFFD REPLACEMENT CHARACTER
// without panicking, and parsing must continue normally afterwards.
func TestUnicodeEscapeOutOfRange(t *testing.T) {
	// \U110000 is one past the maximum valid codepoint U+10FFFF.
	b := newTestBundle(t, "x = { \"\\U110000\" }\nnext = After\n")

	got, errs := format(t, b, "x", nil)
	assert.Equal(t, "�", got, "an out-of-range \\U escape becomes U+FFFD")
	assert.Empty(t, errs)

	// Parsing continued past the broken escape: the next entry is intact.
	got, errs = format(t, b, "next", nil)
	assert.Equal(t, "After", got)
	assert.Empty(t, errs)
}

func TestPrimitiveNumbers(t *testing.T) {
	src := "one     = { 1 }\n" +
		"select  = { 1 ->\n" +
		"   *[0] Zero\n" +
		"    [1] One\n" +
		"}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "one", nil)
	assert.Equal(t, "1", got)
	assert.Empty(t, errs)

	got, errs = format(t, b, "select", nil)
	assert.Equal(t, "One", got)
	assert.Empty(t, errs)
}

func TestPrimitiveSimpleStrings(t *testing.T) {
	src := "foo               = Foo\n" +
		"\n" +
		"placeable-literal = { \"Foo\" } Bar\n" +
		"placeable-message = { foo } Bar\n" +
		"\n" +
		"selector-literal = { \"Foo\" ->\n" +
		"   *[Foo] Member 1\n" +
		"}\n" +
		"\n" +
		"bar =\n" +
		"    .attr = Bar Attribute\n" +
		"\n" +
		"placeable-attr   = { bar.attr }\n" +
		"\n" +
		"-baz = Baz\n" +
		"    .attr = BazAttribute\n" +
		"\n" +
		"selector-attr    = { -baz.attr ->\n" +
		"   *[BazAttribute] Member 3\n" +
		"}\n"
	b := newTestBundle(t, src)

	tests := []struct {
		id   string
		want string
	}{
		{"foo", "Foo"},
		{"placeable-literal", "Foo Bar"},
		{"placeable-message", "Foo Bar"},
		{"selector-literal", "Member 1"},
		{"placeable-attr", "Bar Attribute"},
		{"selector-attr", "Member 3"},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			got, errs := format(t, b, tc.id, nil)
			assert.Equal(t, tc.want, got)
			assert.Empty(t, errs)
		})
	}

	// Attribute value directly.
	msg, _ := b.Message("bar")
	got, errs := b.FormatPattern(msg.Attributes["attr"], nil)
	assert.Equal(t, "Bar Attribute", got)
	assert.Empty(t, errs)
}

func TestPrimitiveComplexStrings(t *testing.T) {
	src := "foo               = Foo\n" +
		"bar               = { foo }Bar\n" +
		"\n" +
		"placeable-message = { bar }Baz\n" +
		"\n" +
		"baz =\n" +
		"    .attr = { bar }BazAttribute\n" +
		"\n" +
		"placeable-attr = { baz.attr }\n" +
		"\n" +
		"selector-attr = { baz.attr ->\n" +
		"    [FooBarBazAttribute] FooBarBaz\n" +
		"   *[other] Other\n" +
		"}\n"
	b := newTestBundle(t, src)

	tests := []struct {
		id   string
		want string
	}{
		{"bar", "FooBar"},
		{"placeable-message", "FooBarBaz"},
		{"placeable-attr", "FooBarBazAttribute"},
		{"selector-attr", "FooBarBaz"},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			got, errs := format(t, b, tc.id, nil)
			assert.Equal(t, tc.want, got)
			assert.Empty(t, errs)
		})
	}

	// Attribute value directly.
	t.Run("attribute directly", func(t *testing.T) {
		msg, _ := b.Message("baz")
		got, errs := b.FormatPattern(msg.Attributes["attr"], nil)
		assert.Equal(t, "FooBarBazAttribute", got)
		assert.Empty(t, errs)
	})
}
