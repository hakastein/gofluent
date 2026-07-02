package fluent_test

import (
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
)

// Ported from isolating_test.js.

// Unicode bidi isolation marks (FSI U+2068, PDI U+2069).
const (
	fsi = "⁨"
	pdi = "⁩"
)

func newIsolatingBundle(t *testing.T) *fluent.Bundle {
	t.Helper()
	src := "foo = Foo\n" +
		"bar = { foo } Bar\n" +
		"baz = { $arg } Baz\n" +
		"qux = { bar } { baz }\n"
	b := fluent.NewBundle("en-US")
	b.AddResource(fluent.NewResource(src))
	return b
}

func TestIsolatesMessageReferences(t *testing.T) {
	b := newIsolatingBundle(t)
	got, err := format(t, b, "bar", nil)
	assert.Equal(t, fsi+"Foo"+pdi+" Bar", got)
	assert.NoError(t, err)
}

func TestIsolatesStringVariables(t *testing.T) {
	b := newIsolatingBundle(t)
	got, err := format(t, b, "baz", map[string]any{"arg": "Arg"})
	assert.Equal(t, fsi+"Arg"+pdi+" Baz", got)
	assert.NoError(t, err)
}

func TestIsolatesNumberVariables(t *testing.T) {
	b := newIsolatingBundle(t)
	got, err := format(t, b, "baz", map[string]any{"arg": 1})
	assert.Equal(t, fsi+"1"+pdi+" Baz", got)
	assert.NoError(t, err)
}

func TestIsolatesComplexInterpolations(t *testing.T) {
	b := newIsolatingBundle(t)
	got, err := format(t, b, "qux", map[string]any{"arg": "Arg"})
	expectedBar := fsi + fsi + "Foo" + pdi + " Bar" + pdi
	expectedBaz := fsi + fsi + "Arg" + pdi + " Baz" + pdi
	assert.Equal(t, expectedBar+" "+expectedBaz, got)
	assert.NoError(t, err)
}

func TestSkipsIsolationSinglePlaceable(t *testing.T) {
	src := "-brand-short-name = Amaya\nfoo = { -brand-short-name }\n"
	b := fluent.NewBundle("en-US")
	b.AddResource(fluent.NewResource(src))
	got, err := format(t, b, "foo", nil)
	assert.Equal(t, "Amaya", got)
	assert.NoError(t, err)
}
