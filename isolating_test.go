package fluent_test

import (
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
)

// Ported from isolating_test.js. FSI = U+2068, PDI = U+2069 (declared in
// bundle_test.go as fsi/pdi).

func newIsolatingBundle(t *testing.T) *fluent.Bundle {
	t.Helper()
	src := "foo = Foo\n" +
		"bar = { foo } Bar\n" +
		"baz = { $arg } Baz\n" +
		"qux = { bar } { baz }\n"
	b := fluent.NewBundle("en-US")
	b.AddResource(mustParse(t, src))
	return b
}

func TestIsolatesMessageReferences(t *testing.T) {
	b := newIsolatingBundle(t)
	got, errs := format(t, b, "bar", nil)
	assert.Equal(t, fsi+"Foo"+pdi+" Bar", got)
	assert.Empty(t, errs)
}

func TestIsolatesStringVariables(t *testing.T) {
	b := newIsolatingBundle(t)
	got, errs := format(t, b, "baz", map[string]any{"arg": "Arg"})
	assert.Equal(t, fsi+"Arg"+pdi+" Baz", got)
	assert.Empty(t, errs)
}

func TestIsolatesNumberVariables(t *testing.T) {
	b := newIsolatingBundle(t)
	got, errs := format(t, b, "baz", map[string]any{"arg": 1})
	assert.Equal(t, fsi+"1"+pdi+" Baz", got)
	assert.Empty(t, errs)
}

func TestIsolatesComplexInterpolations(t *testing.T) {
	b := newIsolatingBundle(t)
	got, errs := format(t, b, "qux", map[string]any{"arg": "Arg"})
	expectedBar := fsi + fsi + "Foo" + pdi + " Bar" + pdi
	expectedBaz := fsi + fsi + "Arg" + pdi + " Baz" + pdi
	assert.Equal(t, expectedBar+" "+expectedBaz, got)
	assert.Empty(t, errs)
}

func TestSkipsIsolationSinglePlaceable(t *testing.T) {
	src := "-brand-short-name = Amaya\nfoo = { -brand-short-name }\n"
	b := fluent.NewBundle("en-US")
	b.AddResource(mustParse(t, src))
	got, errs := format(t, b, "foo", nil)
	assert.Equal(t, "Amaya", got)
	assert.Empty(t, errs)
}
