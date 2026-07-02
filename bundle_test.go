package fluent_test

import (
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared test helpers for the fluent_test package. They drive the package
// through its exported API only.

// newLocaleBundle creates a non-isolating bundle for locale and adds the
// resource, requiring it to load cleanly.
func newLocaleBundle(t *testing.T, locale, src string) *fluent.Bundle {
	t.Helper()
	b := fluent.NewBundle(locale, fluent.WithUseIsolating(false))
	require.NoError(t, b.AddResource(fluent.NewResource(src)), "AddResource errors")
	return b
}

// newTestBundle is newLocaleBundle for the default en-US locale.
func newTestBundle(t *testing.T, src string) *fluent.Bundle {
	t.Helper()
	return newLocaleBundle(t, "en-US", src)
}

// format is a convenience helper: get a message value and format it.
func format(t *testing.T, b *fluent.Bundle, id string, args map[string]any) (string, error) {
	t.Helper()
	msg, ok := b.Message(id)
	require.Truef(t, ok, "message %q not found", id)
	return b.FormatPattern(msg.Value(), args)
}

// errList unwraps a joined error into its constituents, for the tests whose
// contract is the exact number of problems reported (not merely their kind).
// A nil error yields nil; a single, un-joined error yields a one-element slice.
func errList(err error) []error {
	if err == nil {
		return nil
	}
	if u, ok := err.(interface{ Unwrap() []error }); ok {
		return u.Unwrap()
	}
	return []error{err}
}

func TestAddResource(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")

	_, ok := b.Message("foo")
	assert.True(t, ok, "expected message foo")
	// A term must not be reachable as a public message...
	_, ok = b.Message("-bar")
	assert.False(t, ok, "-bar should not be retrievable as a message")
	// ...but it must resolve when referenced as a term.
	b.AddResource(fluent.NewResource("use-bar = { -bar }\n"))
	got, err := format(t, b, "use-bar", nil)
	assert.Equal(t, "Bar", got)
	assert.NoError(t, err)
}

func TestMessagesAndTermsShareName(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")
	b.AddResource(fluent.NewResource("-foo = Private Foo\n"))

	// The message foo and the term -foo coexist: foo stays a message, and the
	// term -foo resolves independently when referenced.
	_, ok := b.Message("foo")
	assert.True(t, ok, "foo should remain a message")

	b.AddResource(fluent.NewResource("use-foo = { -foo }\n"))
	got, err := format(t, b, "use-foo", nil)
	assert.Equal(t, "Private Foo", got)
	assert.NoError(t, err)
}

func TestAllowOverrides(t *testing.T) {
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
	b.AddResource(fluent.NewResource("key = Foo"))

	err := b.AddResource(fluent.NewResource("key = Bar"))
	require.Len(t, errList(err), 1, "expected 1 override error")
	got, _ := format(t, b, "key", nil)
	assert.Equal(t, "Foo", got)

	err = b.AddResourceOverriding(fluent.NewResource("key = Bar"))
	require.NoError(t, err, "expected no errors with overriding")
	got, _ = format(t, b, "key", nil)
	assert.Equal(t, "Bar", got)
}

func TestBrokenEntriesAreNotPublic(t *testing.T) {
	src := "foo = Foo\n" +
		"bar =\n" +
		"    .attr = Bar Attr\n" +
		"-term = Term\n" +
		"\n" +
		"err1 =\n" +
		"err2 = {}\n" +
		"err3 =\n" +
		"    .attr =\n" +
		"err4 =\n" +
		"    .attr1 = Attr\n" +
		"    .attr2 = {}\n"
	b := newTestBundle(t, src)

	_, ok := b.Message("foo")
	assert.True(t, ok, "foo should exist")
	for _, id := range []string{"-term", "missing", "-missing", "err1", "err2", "err3", "err4"} {
		_, ok := b.Message(id)
		assert.Falsef(t, ok, "%q should not be a public message", id)
	}
}

// The runtime parser accepts the same whitespace inside placeables as
// fluent.js, whose token regexes use JavaScript's \s (tabs, NBSP, ...) — not
// just spaces and newlines.
func TestRuntimeParserAcceptsJSWhitespace(t *testing.T) {
	b := newTestBundle(t, "m = {\t$x }\n")
	got, errs := format(t, b, "m", map[string]any{"x": "ok"})
	assert.Equal(t, "ok", got)
	assert.Empty(t, errs)
}

func TestFormatPatternNilPattern(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n")

	// A nil pattern (e.g. the Value of an attribute-only message) must not
	// panic: it renders the {???} fallback with a type error.
	got, err := b.FormatPattern(nil, nil)
	assert.Equal(t, "{???}", got)
	require.ErrorIs(t, err, fluent.ErrType)
}

func TestAttributeOnlyMessage(t *testing.T) {
	b := newTestBundle(t, "bar =\n    .attr = Bar Attr\n")

	msg, ok := b.Message("bar")
	require.True(t, ok, "an attribute-only message is still a public message")
	assert.Nil(t, msg.Value(), "attribute-only message has no value")

	attr, _ := msg.Attribute("attr")
	got, err := b.FormatPattern(attr, nil)
	assert.Equal(t, "Bar Attr", got)
	assert.NoError(t, err)
}

func TestMessageLookup(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")

	msg, ok := b.Message("foo")
	require.True(t, ok, "expected foo")
	assert.Equal(t, "foo", msg.ID())
	require.NotNil(t, msg.Value())
	got, err := b.FormatPattern(msg.Value(), nil)
	assert.Equal(t, "Foo", got)
	assert.NoError(t, err)
	assert.Empty(t, msg.AttributeNames())

	_, ok = b.Message("-bar")
	assert.False(t, ok, "-bar should not be retrievable as a message")
}
