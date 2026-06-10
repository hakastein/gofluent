package langneg_test

import (
	"testing"

	"github.com/hakastein/gofluent/langneg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// negCase is one table row: requested * available [+ default] = supported.
type negCase struct {
	name      string
	requested []string
	available []string
	def       string
	supported []string
}

// The cases below are ported from @fluent/langneg 0.6.2's langneg_test.js (the
// self-contained, pre-Intl.Locale algorithm this package mirrors). An empty
// negotiation result is nil by contract.

func TestNegotiateFiltering(t *testing.T) {
	cases := []negCase{
		// exact match
		{"exact en", []string{"en"}, []string{"en"}, "", []string{"en"}},
		{"exact en-US", []string{"en-US"}, []string{"en-US"}, "", []string{"en-US"}},
		{"exact en-Latn-US", []string{"en-Latn-US"}, []string{"en-Latn-US"}, "", []string{"en-Latn-US"}},
		{"exact en-Latn-US-macos", []string{"en-Latn-US-macos"}, []string{"en-Latn-US-macos"}, "", []string{"en-Latn-US-macos"}},
		{"exact rm-surmiran", []string{"rm-surmiran"}, []string{"rm-surmiran"}, "", []string{"rm-surmiran"}},
		{"exact de-1996", []string{"de-1996"}, []string{"de-1996"}, "", []string{"de-1996"}},
		{"exact fr-FR among many", []string{"fr-FR"}, []string{"de", "it", "fr-FR"}, "", []string{"fr-FR"}},
		{"exact multi", []string{"fr", "pl", "de-DE"}, []string{"pl", "en-US", "de-DE"}, "", []string{"pl", "de-DE"}},

		// available as range
		{"range en-US*en", []string{"en-US"}, []string{"en"}, "", []string{"en"}},
		{"range en-Latn-US*en-US", []string{"en-Latn-US"}, []string{"en-US"}, "", []string{"en-US"}},
		{"range en-US-macos*en-US", []string{"en-US-macos"}, []string{"en-US"}, "", []string{"en-US"}},
		{"range fr-CA,de-DE", []string{"fr-CA", "de-DE"}, []string{"fr", "it", "de"}, "", []string{"fr", "de"}},
		{"range ja-JP-macos*ja", []string{"ja-JP-macos"}, []string{"ja"}, "", []string{"ja"}},
		{"range en-Latn", []string{"en-Latn-GB", "en-Latn-IN"}, []string{"en-IN", "en-GB"}, "", []string{"en-GB", "en-IN"}},

		// likely subtag
		{"likely en", []string{"en"}, []string{"en-GB", "de", "en-US"}, "", []string{"en-US", "en-GB"}},
		{"likely en-Latn", []string{"en"}, []string{"en-Latn-GB", "de", "en-Latn-US"}, "", []string{"en-Latn-US", "en-Latn-GB"}},
		{"likely fr", []string{"fr"}, []string{"fr-CA", "fr-FR"}, "", []string{"fr-FR", "fr-CA"}},
		{"likely az-IR", []string{"az-IR"}, []string{"az-Latn", "az-Arab"}, "", []string{"az-Arab"}},
		{"likely sr-RU", []string{"sr-RU"}, []string{"sr-Cyrl", "sr-Latn"}, "", []string{"sr-Latn"}},
		{"likely sr", []string{"sr"}, []string{"sr-Latn", "sr-Cyrl"}, "", []string{"sr-Cyrl"}},
		{"likely zh-GB", []string{"zh-GB"}, []string{"zh-Hans", "zh-Hant"}, "", []string{"zh-Hant"}},
		{"likely sr,ru", []string{"sr", "ru"}, []string{"sr-Latn", "ru"}, "", []string{"ru"}},
		{"likely sr-RU cross", []string{"sr-RU"}, []string{"sr-Latn-RO", "sr-Cyrl"}, "", []string{"sr-Latn-RO"}},

		// cross-region
		{"cross en*en-US", []string{"en"}, []string{"en-US"}, "", []string{"en-US"}},
		{"cross en-US*en-GB", []string{"en-US"}, []string{"en-GB"}, "", []string{"en-GB"}},
		{"cross en-Latn-US*en-Latn-GB", []string{"en-Latn-US"}, []string{"en-Latn-GB"}, "", []string{"en-Latn-GB"}},

		// cross-variant
		{"cross-variant", []string{"en-US-macos"}, []string{"en-US-windows"}, "", []string{"en-US-windows"}},

		// prioritize
		{"prio exact", []string{"en-US"}, []string{"en-US-macos", "en", "en-US"}, "", []string{"en-US", "en", "en-US-macos"}},
		{"prio range", []string{"en-Latn-US"}, []string{"en-GB", "en-US"}, "", []string{"en-US", "en-GB"}},
		{"prio likely", []string{"en"}, []string{"en-Cyrl-US", "en-Latn-US"}, "", []string{"en-Latn-US"}},
		{"prio variant range", []string{"en-US-macos"}, []string{"en-US-windows", "en-GB-macos"}, "", []string{"en-US-windows", "en-GB-macos"}},
		{"prio regional range", []string{"en-US-macos"}, []string{"en-GB-windows"}, "", []string{"en-GB-windows"}},

		// extra
		{"extra en-US", []string{"en-US"}, []string{"en-GB", "en"}, "", []string{"en", "en-GB"}},
		{"extra zh-HK", []string{"zh-HK"}, []string{"zh-CN", "zh-TW"}, "", []string{"zh-TW", "zh-CN"}},

		// default locale
		{"default none", []string{"fr"}, []string{"de", "it"}, "", nil},
		{"default added", []string{"fr"}, []string{"de", "it"}, "en-US", []string{"en-US"}},
		{"default present", []string{"fr"}, []string{"de", "en-US"}, "en-US", []string{"en-US"}},
		{"default appended", []string{"fr", "de-DE"}, []string{"de-DE", "fr-CA"}, "en-US", []string{"fr-CA", "de-DE", "en-US"}},

		// all matches on 1st higher than any on 2nd
		{"all-first", []string{"fr-CA-macos", "de-DE"}, []string{"de-DE", "fr-FR-windows"}, "", []string{"fr-FR-windows", "de-DE"}},

		// cases and underscores
		{"underscore fr_FR", []string{"fr_FR"}, []string{"fr-FR"}, "", []string{"fr-FR"}},
		{"underscore fr_fr", []string{"fr_fr"}, []string{"fr-fr"}, "", []string{"fr-fr"}},
		{"underscore fr_Fr", []string{"fr_Fr"}, []string{"fr-fR"}, "", []string{"fr-fR"}},
		{"underscore fr_lAtN_fr", []string{"fr_lAtN_fr"}, []string{"fr-Latn-FR"}, "", []string{"fr-Latn-FR"}},
		{"underscore both", []string{"fr_FR"}, []string{"fr_FR"}, "", []string{"fr_FR"}},
		{"underscore mixed", []string{"fr-FR"}, []string{"fr_FR"}, "", []string{"fr_FR"}},
		{"underscore complex", []string{"fr_Cyrl_FR_macos"}, []string{"fr_Cyrl_fr-macos"}, "", []string{"fr_Cyrl_fr-macos"}},

		// invalid input
		{"invalid req", []string{"2"}, []string{"ąóżł"}, "", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := langneg.NegotiateLanguages(c.requested, c.available, c.def, langneg.Filtering)
			require.NoError(t, err)
			assert.Equalf(t, c.supported, got,
				"NegotiateLanguages(%v, %v, %q)", c.requested, c.available, c.def)
		})
	}
}

func TestNegotiateMatching(t *testing.T) {
	got, err := langneg.NegotiateLanguages(
		[]string{"fr", "en"},
		[]string{"en-US", "fr-FR", "en", "fr"},
		"", langneg.Matching)
	require.NoError(t, err)
	assert.Equal(t, []string{"fr", "en"}, got, "matching")
}

func TestNegotiateLookup(t *testing.T) {
	got, err := langneg.NegotiateLanguages(
		[]string{"fr-FR", "en"},
		[]string{"en-US", "fr-FR", "en", "fr"},
		"en-US", langneg.Lookup)
	require.NoError(t, err)
	assert.Equal(t, []string{"fr-FR"}, got, "lookup")
}

func TestLookupDefaultFallback(t *testing.T) {
	got, err := langneg.NegotiateLanguages([]string{"de"}, []string{"fr", "it"}, "en-US", langneg.Lookup)
	require.NoError(t, err)
	assert.Equal(t, []string{"en-US"}, got, "lookup fallback")
}

func TestLookupNeedsDefault(t *testing.T) {
	_, err := langneg.NegotiateLanguages([]string{"de"}, []string{"fr"}, "", langneg.Lookup)
	require.ErrorIs(t, err, langneg.ErrLookupNeedsDefault)
}

func TestAcceptedLanguages(t *testing.T) {
	cases := []struct {
		header string
		want   []string
	}{
		{"en-US, fr, pl", []string{"en-US", "fr", "pl"}},
		{"sr-Latn", []string{"sr-Latn"}},
		{"fr-CH, fr;q=0.9, en;q=0.8, de;q=0.7, *;q=0.5", []string{"fr-CH", "fr", "en", "de", "*"}},
		{"en;q=0.8, fr;q=0.9, de;q=0.7, *;q=0.5, fr-CH", []string{"fr-CH", "fr", "en", "de", "*"}},
		{"en;q=0.1, fr;q=0.1, de;q=0.1, *;q=0.1", []string{"en", "fr", "de", "*"}},
		{"en;q=0.8,,, fr;q=0.9,, de;q=0.7, *;q=0.5, fr-CH", []string{"fr-CH", "fr", "en", "de", "*"}},
		{"", nil},
	}
	for _, c := range cases {
		got := langneg.AcceptedLanguages(c.header)
		assert.Equalf(t, c.want, got, "AcceptedLanguages(%q)", c.header)
	}
}

func TestLocaleParsing(t *testing.T) {
	cases := []struct {
		in   string
		want langneg.Locale
	}{
		{"en", langneg.Locale{Wellformed: true, Language: "en"}},
		{"lij", langneg.Locale{Wellformed: true, Language: "lij"}},
		{"en-Latn", langneg.Locale{Wellformed: true, Language: "en", Script: "Latn"}},
		{"en-Latn-US", langneg.Locale{Wellformed: true, Language: "en", Script: "Latn", Region: "US"}},
		{"en-Latn-US-macos", langneg.Locale{Wellformed: true, Language: "en", Script: "Latn", Region: "US", Variant: "macos"}},
		{"en-US", langneg.Locale{Wellformed: true, Language: "en", Region: "US"}},
		{"lij-FA-linux", langneg.Locale{Wellformed: true, Language: "lij", Region: "FA", Variant: "linux"}},
	}
	for _, c := range cases {
		got := langneg.NewLocale(c.in)
		assert.Equalf(t, c.want, *got, "NewLocale(%q)", c.in)
	}
}
