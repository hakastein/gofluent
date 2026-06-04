package localization_test

import (
	"testing"
	"testing/fstest"

	fluent "github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/localization"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustBundle builds a bundle for locale from FTL source, failing the test on
// resource/parse errors.
func mustBundle(t *testing.T, locale, source string) *fluent.Bundle {
	t.Helper()
	b := fluent.NewBundle(locale)
	res, errs := fluent.NewResource(source)
	require.Emptyf(t, errs, "parse errors for %s", locale)
	require.Emptyf(t, b.AddResource(res), "add errors for %s", locale)
	return b
}

func TestFallbackChain(t *testing.T) {
	de := mustBundle(t, "de", `
known-de = Hallo aus de
shared = Geteilt de
`)
	en := mustBundle(t, "en", `
known = Hello from en
shared = Shared en
`)

	loc := localization.New([]*fluent.Bundle{de, en})

	// "known" exists only in en -> falls through to en.
	got, errs := loc.FormatValue("known", nil)
	assert.Equal(t, "Hello from en", got, "known")
	assert.Empty(t, errs, "known errs")

	// "known-de" exists in de (highest priority) -> de wins.
	got, errs = loc.FormatValue("known-de", nil)
	assert.Equal(t, "Hallo aus de", got, "known-de")
	assert.Empty(t, errs, "known-de errs")

	// "shared" exists in both -> de wins (first in chain).
	got, _ = loc.FormatValue("shared", nil)
	assert.Equal(t, "Geteilt de", got, "shared")

	// Totally missing -> returns the id plus an error.
	got, errs = loc.FormatValue("nope", nil)
	assert.Equal(t, "nope", got, "missing id returned value")
	require.Len(t, errs, 1, "missing id errs")
	var notFound *localization.NotFoundError
	assert.ErrorAs(t, errs[0], &notFound, "missing id err type")
}

func TestAttributeAccess(t *testing.T) {
	en := mustBundle(t, "en", `
login = Sign in
    .title = Click to sign in
    .aria = Sign-in button
`)
	loc := localization.New([]*fluent.Bundle{en})

	got, errs := loc.FormatValue("login.title", nil)
	assert.Equal(t, "Click to sign in", got, "login.title")
	assert.Empty(t, errs, "login.title errs")

	// Value access on the same message.
	got, _ = loc.FormatValue("login", nil)
	assert.Equal(t, "Sign in", got, "login")

	// Missing attribute -> id returned plus NotFoundError.
	got, errs = loc.FormatValue("login.missing", nil)
	assert.Equal(t, "login.missing", got, "login.missing value")
	assert.Len(t, errs, 1, "login.missing errs")
}

func TestAttributeFallsThrough(t *testing.T) {
	// de has the message but not the attribute; en has the attribute.
	de := mustBundle(t, "de", `greeting = Hallo`)
	en := mustBundle(t, "en", `
greeting = Hello
    .tooltip = A friendly greeting
`)
	loc := localization.New([]*fluent.Bundle{de, en})

	got, errs := loc.FormatValue("greeting.tooltip", nil)
	assert.Equal(t, "A friendly greeting", got, "greeting.tooltip want en attribute")
	assert.Empty(t, errs, "greeting.tooltip errs")
}

func TestFormatValueWithArgs(t *testing.T) {
	en := mustBundle(t, "en", `welcome = Welcome, { $name }!`)
	loc := localization.New([]*fluent.Bundle{en})
	got, errs := loc.FormatValue("welcome", map[string]any{"name": "Mary"})
	assert.Equal(t, "Welcome, ⁨Mary⁩!", got, "welcome")
	assert.Empty(t, errs, "welcome errs")
}

func TestFormatMessage(t *testing.T) {
	en := mustBundle(t, "en", `
btn = Save
    .title = Save the document
`)
	loc := localization.New([]*fluent.Bundle{en})

	value, attrs, found, errs := loc.FormatMessage("btn", nil)
	require.True(t, found, "FormatMessage btn: not found")
	assert.Equal(t, "Save", value, "value")
	assert.Equal(t, "Save the document", attrs["title"], "attrs[title]")
	assert.Empty(t, errs, "unexpected errs")

	_, _, found, errs = loc.FormatMessage("ghost", nil)
	assert.False(t, found, "ghost should not be found")
	assert.Len(t, errs, 1, "ghost errs")
}

func TestFormatValues(t *testing.T) {
	en := mustBundle(t, "en", `
a = Apple
b = Banana
`)
	loc := localization.New([]*fluent.Bundle{en})
	got, errs := loc.FormatValues([]localization.L10nID{
		{ID: "a"},
		{ID: "b"},
		{ID: "missing"},
	})
	assert.Equal(t, []string{"Apple", "Banana", "missing"}, got, "FormatValues")
	assert.Len(t, errs, 1, "errs want one (for missing)")
}

// TestIntegrationFSNegotiation exercises the full path: an in-memory fs.FS
// loader plus langneg negotiation produce a real fallback chain, which then
// formats a value end-to-end with fallback.
func TestIntegrationFSNegotiation(t *testing.T) {
	fsys := fstest.MapFS{
		"de/main.ftl": {Data: []byte("hello-de = Hallo\nshared = Geteilt\n")},
		"en/main.ftl": {Data: []byte("hello = Hello\nshared = Shared\nonly-en = Only English\n")},
	}

	loc, errs := localization.NewFromLocales(
		[]string{"de-AT", "en-US"}, // requested
		[]string{"de", "en"},       // available
		"en",                       // default
		[]string{"main"},           // resource ids
		localization.FSLoader(fsys, "{locale}/{resource}.ftl"),
	)
	require.Empty(t, errs, "NewFromLocales errors")

	// Negotiation should produce the chain [de, en].
	bundles := loc.Bundles()
	locales := make([]string, len(bundles))
	for i, b := range bundles {
		locales[i] = b.Locale()
	}
	require.Equal(t, []string{"de", "en"}, locales, "negotiated chain")

	// "shared" -> de wins.
	got, _ := loc.FormatValue("shared", nil)
	assert.Equal(t, "Geteilt", got, "shared")
	// "only-en" -> falls through to en.
	got, errs = loc.FormatValue("only-en", nil)
	assert.Equal(t, "Only English", got, "only-en")
	assert.Empty(t, errs, "only-en errs")
	// "hello-de" -> only in de.
	got, _ = loc.FormatValue("hello-de", nil)
	assert.Equal(t, "Hallo", got, "hello-de")
}

// TestFSLoaderMissingResourceTolerant verifies a missing resource for one
// locale does not abort building the chain.
func TestFSLoaderMissingResourceTolerant(t *testing.T) {
	fsys := fstest.MapFS{
		"en/main.ftl": {Data: []byte("hi = Hi\n")},
		// de/main.ftl intentionally absent.
	}
	loc, errs := localization.NewFromLocales(
		[]string{"de", "en"},
		[]string{"de", "en"},
		"en",
		[]string{"main"},
		localization.FSLoader(fsys, "{locale}/{resource}.ftl"),
	)
	// de bundle load fails -> recorded as an error, but chain still built.
	assert.NotEmpty(t, errs, "expected a load error for the missing de resource")
	got, _ := loc.FormatValue("hi", nil)
	assert.Equal(t, "Hi", got, "hi want \"Hi\" (en still loaded)")
}
