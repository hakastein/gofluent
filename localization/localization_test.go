package localization

import (
	"testing"
	"testing/fstest"

	fluent "github.com/hakastein/gofluent"
)

// mustBundle builds a bundle for locale from FTL source, failing the test on
// resource/parse errors.
func mustBundle(t *testing.T, locale, source string) *fluent.Bundle {
	t.Helper()
	b := fluent.NewBundle(locale)
	res, errs := fluent.NewResource(source)
	if len(errs) > 0 {
		t.Fatalf("parse errors for %s: %v", locale, errs)
	}
	if addErrs := b.AddResource(res); len(addErrs) > 0 {
		t.Fatalf("add errors for %s: %v", locale, addErrs)
	}
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

	loc := New([]*fluent.Bundle{de, en})

	// "known" exists only in en -> falls through to en.
	if got, errs := loc.FormatValue("known", nil); got != "Hello from en" || len(errs) > 0 {
		t.Errorf("known = %q, errs %v; want \"Hello from en\"", got, errs)
	}

	// "known-de" exists in de (highest priority) -> de wins.
	if got, errs := loc.FormatValue("known-de", nil); got != "Hallo aus de" || len(errs) > 0 {
		t.Errorf("known-de = %q, errs %v; want \"Hallo aus de\"", got, errs)
	}

	// "shared" exists in both -> de wins (first in chain).
	if got, _ := loc.FormatValue("shared", nil); got != "Geteilt de" {
		t.Errorf("shared = %q; want \"Geteilt de\"", got)
	}

	// Totally missing -> returns the id plus an error.
	got, errs := loc.FormatValue("nope", nil)
	if got != "nope" {
		t.Errorf("missing id returned %q; want \"nope\"", got)
	}
	if len(errs) != 1 {
		t.Fatalf("missing id errs = %v; want exactly one", errs)
	}
	if _, ok := errs[0].(*NotFoundError); !ok {
		t.Errorf("missing id err type = %T; want *NotFoundError", errs[0])
	}
}

func TestAttributeAccess(t *testing.T) {
	en := mustBundle(t, "en", `
login = Sign in
    .title = Click to sign in
    .aria = Sign-in button
`)
	loc := New([]*fluent.Bundle{en})

	if got, errs := loc.FormatValue("login.title", nil); got != "Click to sign in" || len(errs) > 0 {
		t.Errorf("login.title = %q, errs %v; want \"Click to sign in\"", got, errs)
	}

	// Value access on the same message.
	if got, _ := loc.FormatValue("login", nil); got != "Sign in" {
		t.Errorf("login = %q; want \"Sign in\"", got)
	}

	// Missing attribute -> id returned plus NotFoundError.
	got, errs := loc.FormatValue("login.missing", nil)
	if got != "login.missing" || len(errs) != 1 {
		t.Errorf("login.missing = %q errs %v; want id + 1 error", got, errs)
	}
}

func TestAttributeFallsThrough(t *testing.T) {
	// de has the message but not the attribute; en has the attribute.
	de := mustBundle(t, "de", `greeting = Hallo`)
	en := mustBundle(t, "en", `
greeting = Hello
    .tooltip = A friendly greeting
`)
	loc := New([]*fluent.Bundle{de, en})

	if got, errs := loc.FormatValue("greeting.tooltip", nil); got != "A friendly greeting" || len(errs) > 0 {
		t.Errorf("greeting.tooltip = %q errs %v; want en attribute", got, errs)
	}
}

func TestFormatValueWithArgs(t *testing.T) {
	en := mustBundle(t, "en", `welcome = Welcome, { $name }!`)
	loc := New([]*fluent.Bundle{en})
	got, errs := loc.FormatValue("welcome", map[string]any{"name": "Mary"})
	if got != "Welcome, ⁨Mary⁩!" || len(errs) > 0 {
		t.Errorf("welcome = %q errs %v", got, errs)
	}
}

func TestFormatMessage(t *testing.T) {
	en := mustBundle(t, "en", `
btn = Save
    .title = Save the document
`)
	loc := New([]*fluent.Bundle{en})

	value, attrs, found, errs := loc.FormatMessage("btn", nil)
	if !found {
		t.Fatal("FormatMessage btn: not found")
	}
	if value != "Save" {
		t.Errorf("value = %q; want \"Save\"", value)
	}
	if attrs["title"] != "Save the document" {
		t.Errorf("attrs[title] = %q; want \"Save the document\"", attrs["title"])
	}
	if len(errs) > 0 {
		t.Errorf("unexpected errs %v", errs)
	}

	_, _, found, errs = loc.FormatMessage("ghost", nil)
	if found {
		t.Error("ghost should not be found")
	}
	if len(errs) != 1 {
		t.Errorf("ghost errs = %v; want one", errs)
	}
}

func TestFormatValues(t *testing.T) {
	en := mustBundle(t, "en", `
a = Apple
b = Banana
`)
	loc := New([]*fluent.Bundle{en})
	got, errs := loc.FormatValues([]L10nID{
		{ID: "a"},
		{ID: "b"},
		{ID: "missing"},
	})
	if len(got) != 3 || got[0] != "Apple" || got[1] != "Banana" || got[2] != "missing" {
		t.Errorf("FormatValues = %v", got)
	}
	if len(errs) != 1 {
		t.Errorf("errs = %v; want one (for missing)", errs)
	}
}

// TestIntegrationFSNegotiation exercises the full path: an in-memory fs.FS
// loader plus langneg negotiation produce a real fallback chain, which then
// formats a value end-to-end with fallback.
func TestIntegrationFSNegotiation(t *testing.T) {
	fsys := fstest.MapFS{
		"de/main.ftl": {Data: []byte("hello-de = Hallo\nshared = Geteilt\n")},
		"en/main.ftl": {Data: []byte("hello = Hello\nshared = Shared\nonly-en = Only English\n")},
	}

	loc, errs := NewFromLocales(
		[]string{"de-AT", "en-US"}, // requested
		[]string{"de", "en"},       // available
		"en",                       // default
		[]string{"main"},           // resource ids
		FSLoader(fsys, "{locale}/{resource}.ftl"),
	)
	if len(errs) > 0 {
		t.Fatalf("NewFromLocales errors: %v", errs)
	}

	// Negotiation should produce the chain [de, en].
	bundles := loc.Bundles()
	if len(bundles) != 2 || bundles[0].Locale() != "de" || bundles[1].Locale() != "en" {
		var locales []string
		for _, b := range bundles {
			locales = append(locales, b.Locale())
		}
		t.Fatalf("negotiated chain = %v; want [de en]", locales)
	}

	// "shared" -> de wins.
	if got, _ := loc.FormatValue("shared", nil); got != "Geteilt" {
		t.Errorf("shared = %q; want \"Geteilt\"", got)
	}
	// "only-en" -> falls through to en.
	if got, errs := loc.FormatValue("only-en", nil); got != "Only English" || len(errs) > 0 {
		t.Errorf("only-en = %q errs %v; want \"Only English\"", got, errs)
	}
	// "hello-de" -> only in de.
	if got, _ := loc.FormatValue("hello-de", nil); got != "Hallo" {
		t.Errorf("hello-de = %q; want \"Hallo\"", got)
	}
}

// TestFSLoaderMissingResourceTolerant verifies a missing resource for one
// locale does not abort building the chain.
func TestFSLoaderMissingResourceTolerant(t *testing.T) {
	fsys := fstest.MapFS{
		"en/main.ftl": {Data: []byte("hi = Hi\n")},
		// de/main.ftl intentionally absent.
	}
	loc, errs := NewFromLocales(
		[]string{"de", "en"},
		[]string{"de", "en"},
		"en",
		[]string{"main"},
		FSLoader(fsys, "{locale}/{resource}.ftl"),
	)
	// de bundle load fails -> recorded as an error, but chain still built.
	if len(errs) == 0 {
		t.Error("expected a load error for the missing de resource")
	}
	if got, _ := loc.FormatValue("hi", nil); got != "Hi" {
		t.Errorf("hi = %q; want \"Hi\" (en still loaded)", got)
	}
}
