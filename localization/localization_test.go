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
// resource errors.
func mustBundle(t *testing.T, locale, source string) *fluent.Bundle {
	t.Helper()
	b := fluent.NewBundle(locale)
	require.NoErrorf(t, b.AddResource(fluent.NewResource(source)), "add errors for %s", locale)
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
	got, err := loc.FormatValue("known", nil)
	assert.Equal(t, "Hello from en", got, "known")
	assert.NoError(t, err, "known errs")

	// "known-de" exists in de (highest priority) -> de wins.
	got, err = loc.FormatValue("known-de", nil)
	assert.Equal(t, "Hallo aus de", got, "known-de")
	assert.NoError(t, err, "known-de errs")

	// "shared" exists in both -> de wins (first in chain).
	got, _ = loc.FormatValue("shared", nil)
	assert.Equal(t, "Geteilt de", got, "shared")

	// Totally missing -> returns the id plus an error.
	got, err = loc.FormatValue("nope", nil)
	assert.Equal(t, "nope", got, "missing id returned value")
	var notFound *localization.NotFoundError
	assert.ErrorAs(t, err, &notFound, "missing id err type")
}

func TestAttributeAccess(t *testing.T) {
	en := mustBundle(t, "en", `
login = Sign in
    .title = Click to sign in
    .aria = Sign-in button
`)
	loc := localization.New([]*fluent.Bundle{en})

	got, err := loc.FormatValue("login.title", nil)
	assert.Equal(t, "Click to sign in", got, "login.title")
	assert.NoError(t, err, "login.title errs")

	// Value access on the same message.
	got, _ = loc.FormatValue("login", nil)
	assert.Equal(t, "Sign in", got, "login")

	// Missing attribute -> id returned plus NotFoundError.
	got, err = loc.FormatValue("login.missing", nil)
	assert.Equal(t, "login.missing", got, "login.missing value")
	assert.Error(t, err, "login.missing errs")
}

func TestAttributeFallsThrough(t *testing.T) {
	// de has the message but not the attribute; en has the attribute.
	de := mustBundle(t, "de", `greeting = Hallo`)
	en := mustBundle(t, "en", `
greeting = Hello
    .tooltip = A friendly greeting
`)
	loc := localization.New([]*fluent.Bundle{de, en})

	got, err := loc.FormatValue("greeting.tooltip", nil)
	assert.Equal(t, "A friendly greeting", got, "greeting.tooltip want en attribute")
	assert.NoError(t, err, "greeting.tooltip errs")
}

func TestFormatValueWithArgs(t *testing.T) {
	en := mustBundle(t, "en", `welcome = Welcome, { $name }!`)
	loc := localization.New([]*fluent.Bundle{en})
	got, err := loc.FormatValue("welcome", map[string]any{"name": "Mary"})
	assert.Equal(t, "Welcome, ⁨Mary⁩!", got, "welcome")
	assert.NoError(t, err, "welcome errs")
}

func TestFormatMessage(t *testing.T) {
	en := mustBundle(t, "en", `
btn = Save
    .title = Save the document
`)
	loc := localization.New([]*fluent.Bundle{en})

	msg, err := loc.FormatMessage("btn", nil)
	assert.Equal(t, "Save", msg.Value, "value")
	assert.Equal(t, "Save the document", msg.Attributes["title"], "attrs[title]")
	assert.NoError(t, err, "unexpected errs")

	msg, err = loc.FormatMessage("ghost", nil)
	assert.Equal(t, "ghost", msg.Value, "missing id is returned as the value")
	var notFound *localization.NotFoundError
	assert.ErrorAs(t, err, &notFound, "ghost err type")
}

// TestIntegrationFSNegotiation exercises the full path: an in-memory fs.FS
// loader plus langneg negotiation produce a real fallback chain, which then
// formats a value end-to-end with fallback.
func TestIntegrationFSNegotiation(t *testing.T) {
	fsys := fstest.MapFS{
		"de/main.ftl": {Data: []byte("hello-de = Hallo\nshared = Geteilt\n")},
		"en/main.ftl": {Data: []byte("hello = Hello\nshared = Shared\nonly-en = Only English\n")},
	}

	loc, err := localization.NewFromLocales(localization.Config{
		Requested: []string{"de-AT", "en-US"},
		Available: []string{"de", "en"},
		Default:   "en",
		Resources: []string{"main"},
		Loader:    localization.FSLoader(fsys, "{locale}/{resource}.ftl"),
	})
	require.NoError(t, err, "NewFromLocales errors")

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
	got, err = loc.FormatValue("only-en", nil)
	assert.Equal(t, "Only English", got, "only-en")
	assert.NoError(t, err, "only-en errs")
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
	loc, err := localization.NewFromLocales(localization.Config{
		Requested: []string{"de", "en"},
		Available: []string{"de", "en"},
		Default:   "en",
		Resources: []string{"main"},
		Loader:    localization.FSLoader(fsys, "{locale}/{resource}.ftl"),
	})
	// de bundle load fails -> recorded as an error, but chain still built.
	assert.Error(t, err, "expected a load error for the missing de resource")
	got, _ := loc.FormatValue("hi", nil)
	assert.Equal(t, "Hi", got, "hi want \"Hi\" (en still loaded)")
}
