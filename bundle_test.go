package fluent_test

import (
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared test helpers for the fluent_test package. They drive the package
// through its exported API only.

// newTestBundle creates a bundle with useIsolating=false and adds the resource.
func newTestBundle(t *testing.T, src string) *fluent.Bundle {
	t.Helper()
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
	b.AddResource(fluent.NewResource(src))
	return b
}

// format is a convenience helper: get a message value and format it.
func format(t *testing.T, b *fluent.Bundle, id string, args map[string]any) (string, []error) {
	t.Helper()
	msg, ok := b.Message(id)
	require.Truef(t, ok, "message %q not found", id)
	return b.FormatPattern(msg.Value, args)
}

// intPtr is local test scaffolding for building NumberOptions pointer fields; it
// references no production symbol.
func intPtr(i int) *int { return &i }

func TestAddResource(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")

	_, ok := b.Message("foo")
	assert.True(t, ok, "expected message foo")
	// A term must not be reachable as a public message...
	_, ok = b.Message("-bar")
	assert.False(t, ok, "-bar should not be retrievable as a message")
	// ...but it must resolve when referenced as a term.
	b.AddResource(fluent.NewResource("use-bar = { -bar }\n"))
	got, errs := format(t, b, "use-bar", nil)
	assert.Equal(t, "Bar", got)
	assert.Empty(t, errs)
}

func TestMessagesAndTermsShareName(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")
	b.AddResource(fluent.NewResource("-foo = Private Foo\n"))

	// The message foo and the term -foo coexist: foo stays a message, and the
	// term -foo resolves independently when referenced.
	_, ok := b.Message("foo")
	assert.True(t, ok, "foo should remain a message")

	b.AddResource(fluent.NewResource("use-foo = { -foo }\n"))
	got, errs := format(t, b, "use-foo", nil)
	assert.Equal(t, "Private Foo", got)
	assert.Empty(t, errs)
}

func TestAllowOverrides(t *testing.T) {
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
	b.AddResource(fluent.NewResource("key = Foo"))

	errs := b.AddResource(fluent.NewResource("key = Bar"))
	require.Len(t, errs, 1, "expected 1 override error")
	got, _ := format(t, b, "key", nil)
	assert.Equal(t, "Foo", got)

	errs = b.AddResourceOverriding(fluent.NewResource("key = Bar"))
	require.Empty(t, errs, "expected no errors with overriding")
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

func TestFormatPatternNilPattern(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n")

	// A nil pattern (e.g. the Value of an attribute-only message) must not
	// panic: it renders the {???} fallback with a type error.
	got, errs := b.FormatPattern(nil, nil)
	assert.Equal(t, "{???}", got)
	require.Len(t, errs, 1)
	require.ErrorIs(t, errs[0], fluent.ErrType)
}

func TestAttributeOnlyMessage(t *testing.T) {
	b := newTestBundle(t, "bar =\n    .attr = Bar Attr\n")

	msg, ok := b.Message("bar")
	require.True(t, ok, "an attribute-only message is still a public message")
	assert.Nil(t, msg.Value, "attribute-only message has no value")

	got, errs := b.FormatPattern(msg.Attributes["attr"], nil)
	assert.Equal(t, "Bar Attr", got)
	assert.Empty(t, errs)
}

func TestMessageLookup(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")

	msg, ok := b.Message("foo")
	require.True(t, ok, "expected foo")
	assert.Equal(t, "foo", msg.ID)
	require.NotNil(t, msg.Value)
	got, errs := b.FormatPattern(msg.Value, nil)
	assert.Equal(t, "Foo", got)
	assert.Empty(t, errs)
	assert.Empty(t, msg.Attributes)

	_, ok = b.Message("-bar")
	assert.False(t, ok, "-bar should not be retrievable as a message")
}
