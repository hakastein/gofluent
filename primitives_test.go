package fluent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Ported from primitives_test.js.

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
	msg, _ := b.GetMessage("bar")
	var aerr []error
	assert.Equal(t, "Bar Attribute", b.FormatPattern(msg.Attributes["attr"], nil, &aerr))
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

	got, errs := format(t, b, "bar", nil)
	assert.Equal(t, "FooBar", got)
	assert.Empty(t, errs)

	got, errs = format(t, b, "placeable-message", nil)
	assert.Equal(t, "FooBarBaz", got)
	assert.Empty(t, errs)

	msg, _ := b.GetMessage("baz")
	var aerr []error
	assert.Equal(t, "FooBarBazAttribute", b.FormatPattern(msg.Attributes["attr"], nil, &aerr))

	got, errs = format(t, b, "placeable-attr", nil)
	assert.Equal(t, "FooBarBazAttribute", got)
	assert.Empty(t, errs)

	got, errs = format(t, b, "selector-attr", nil)
	assert.Equal(t, "FooBarBaz", got)
	assert.Empty(t, errs)
}
