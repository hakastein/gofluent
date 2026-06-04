package fluent_test

import (
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared test helpers for the fluent_test package. They drive the package
// through its exported API only.

// mustParse parses FTL and fails the test on a hard parse error.
func mustParse(t *testing.T, src string) *fluent.Resource {
	t.Helper()
	res, errs := fluent.NewResource(src)
	require.Empty(t, errs, "NewResource returned errors")
	return res
}

// newTestBundle creates a bundle with useIsolating=false and adds the resource.
func newTestBundle(t *testing.T, src string) *fluent.Bundle {
	t.Helper()
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
	b.AddResource(mustParse(t, src))
	return b
}

// format is a convenience helper: get a message value and format it.
func format(t *testing.T, b *fluent.Bundle, id string, args map[string]any) (string, []error) {
	t.Helper()
	msg, ok := b.GetMessage(id)
	require.Truef(t, ok, "message %q not found", id)
	var errs []error
	val := b.FormatPatternAny(msg.Value, args, &errs)
	return val, errs
}

// intPtr is local test scaffolding for building NumberOptions pointer fields; it
// references no production symbol.
func intPtr(i int) *int { return &i }

// fsi / pdi are the Unicode bidi isolation marks the bundle wraps placeables in
// when useIsolating is enabled (FSI = U+2068, PDI = U+2069). Declared locally so
// isolation tests assert against the public rendering without reaching into the
// package's unexported constants.
const (
	fsi = "⁨"
	pdi = "⁩"
)

func TestAddResource(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")

	assert.True(t, b.HasMessage("foo"), "expected message foo")
	// A term must not be reachable as a public message...
	assert.False(t, b.HasMessage("-bar"), "-bar should not be a message")
	_, ok := b.GetMessage("-bar")
	assert.False(t, ok, "-bar should not be retrievable as a message")
	// ...but it must resolve when referenced as a term.
	b.AddResource(mustParse(t, "use-bar = { -bar }\n"))
	got, errs := format(t, b, "use-bar", nil)
	assert.Equal(t, "Bar", got)
	assert.Empty(t, errs)
}

func TestMessagesAndTermsShareName(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")
	b.AddResource(mustParse(t, "-foo = Private Foo\n"))

	// The message foo and the term -foo coexist: foo stays a message, and the
	// term -foo resolves independently when referenced.
	assert.True(t, b.HasMessage("foo"), "foo should remain a message")

	b.AddResource(mustParse(t, "use-foo = { -foo }\n"))
	got, errs := format(t, b, "use-foo", nil)
	assert.Equal(t, "Private Foo", got)
	assert.Empty(t, errs)
}

func TestAllowOverrides(t *testing.T) {
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
	b.AddResource(mustParse(t, "key = Foo"))

	errs := b.AddResource(mustParse(t, "key = Bar"))
	require.Len(t, errs, 1, "expected 1 override error")
	msg, _ := b.GetMessage("key")
	assert.Equal(t, "Foo", b.FormatPattern(msg.Value, nil, nil))

	errs = b.AddResourceOverriding(mustParse(t, "key = Bar"))
	require.Empty(t, errs, "expected no errors with overriding")
	msg, _ = b.GetMessage("key")
	assert.Equal(t, "Bar", b.FormatPattern(msg.Value, nil, nil))
}

func TestHasMessageBrokenEntries(t *testing.T) {
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

	assert.True(t, b.HasMessage("foo"), "foo should exist")
	for _, id := range []string{"-term", "missing", "-missing", "err1", "err2", "err3", "err4"} {
		assert.Falsef(t, b.HasMessage(id), "%q should not be a public message", id)
	}
}

func TestGetMessageReturnsValue(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")

	msg, ok := b.GetMessage("foo")
	require.True(t, ok, "expected foo")
	assert.Equal(t, "foo", msg.ID)
	assert.Equal(t, "Foo", msg.Value)
	assert.Empty(t, msg.Attributes)

	_, ok = b.GetMessage("-bar")
	assert.False(t, ok, "-bar should not be retrievable as a message")
}
