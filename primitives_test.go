package fluent

import "testing"

// Ported from primitives_test.js.

func TestPrimitiveNumbers(t *testing.T) {
	src := "one     = { 1 }\n" +
		"select  = { 1 ->\n" +
		"   *[0] Zero\n" +
		"    [1] One\n" +
		"}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "one", nil)
	if got != "1" || len(errs) != 0 {
		t.Errorf("one: %q %v", got, errs)
	}
	got, errs = format(t, b, "select", nil)
	if got != "One" || len(errs) != 0 {
		t.Errorf("select: %q %v", got, errs)
	}
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
		got, errs := format(t, b, tc.id, nil)
		if got != tc.want || len(errs) != 0 {
			t.Errorf("%s: got %q want %q errs %v", tc.id, got, tc.want, errs)
		}
	}

	// Attribute value directly.
	msg, _ := b.GetMessage("bar")
	var aerr []error
	if got := b.FormatPattern(msg.Attributes["attr"], nil, &aerr); got != "Bar Attribute" {
		t.Errorf("bar.attr: %q", got)
	}
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
	if got != "FooBar" || len(errs) != 0 {
		t.Errorf("bar: %q %v", got, errs)
	}
	got, errs = format(t, b, "placeable-message", nil)
	if got != "FooBarBaz" || len(errs) != 0 {
		t.Errorf("placeable-message: %q %v", got, errs)
	}
	msg, _ := b.GetMessage("baz")
	var aerr []error
	if got := b.FormatPattern(msg.Attributes["attr"], nil, &aerr); got != "FooBarBazAttribute" {
		t.Errorf("baz.attr: %q", got)
	}
	got, errs = format(t, b, "placeable-attr", nil)
	if got != "FooBarBazAttribute" || len(errs) != 0 {
		t.Errorf("placeable-attr: %q %v", got, errs)
	}
	got, errs = format(t, b, "selector-attr", nil)
	if got != "FooBarBaz" || len(errs) != 0 {
		t.Errorf("selector-attr: %q %v", got, errs)
	}
}
