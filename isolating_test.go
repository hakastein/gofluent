package fluent

import "testing"

// Ported from isolating_test.js. FSI = U+2068, PDI = U+2069.

func newIsolatingBundle(t *testing.T) *Bundle {
	t.Helper()
	src := "foo = Foo\n" +
		"bar = { foo } Bar\n" +
		"baz = { $arg } Baz\n" +
		"qux = { bar } { baz }\n"
	b := NewBundle("en-US")
	b.AddResource(mustParse(t, src))
	return b
}

func TestIsolatesMessageReferences(t *testing.T) {
	b := newIsolatingBundle(t)
	got, errs := format(t, b, "bar", nil)
	want := fsi + "Foo" + pdi + " Bar"
	if got != want || len(errs) != 0 {
		t.Errorf("got %q want %q errs %v", got, want, errs)
	}
}

func TestIsolatesStringVariables(t *testing.T) {
	b := newIsolatingBundle(t)
	got, errs := format(t, b, "baz", map[string]any{"arg": "Arg"})
	want := fsi + "Arg" + pdi + " Baz"
	if got != want || len(errs) != 0 {
		t.Errorf("got %q want %q errs %v", got, want, errs)
	}
}

func TestIsolatesNumberVariables(t *testing.T) {
	b := newIsolatingBundle(t)
	got, errs := format(t, b, "baz", map[string]any{"arg": 1})
	want := fsi + "1" + pdi + " Baz"
	if got != want || len(errs) != 0 {
		t.Errorf("got %q want %q errs %v", got, want, errs)
	}
}

func TestIsolatesComplexInterpolations(t *testing.T) {
	b := newIsolatingBundle(t)
	got, errs := format(t, b, "qux", map[string]any{"arg": "Arg"})
	expectedBar := fsi + fsi + "Foo" + pdi + " Bar" + pdi
	expectedBaz := fsi + fsi + "Arg" + pdi + " Baz" + pdi
	want := expectedBar + " " + expectedBaz
	if got != want || len(errs) != 0 {
		t.Errorf("got %q want %q errs %v", got, want, errs)
	}
}

func TestSkipsIsolationSinglePlaceable(t *testing.T) {
	src := "-brand-short-name = Amaya\nfoo = { -brand-short-name }\n"
	b := NewBundle("en-US")
	b.AddResource(mustParse(t, src))
	got, errs := format(t, b, "foo", nil)
	if got != "Amaya" || len(errs) != 0 {
		t.Errorf("got %q want Amaya errs %v", got, errs)
	}
}
