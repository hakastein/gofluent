package langneg

import (
	"regexp"
	"strings"
)

// Locale parsing regexp pieces, mirroring locale.ts. A locale id is split into
// up to four fields: language, script, region, variant. Any field may be the
// range character "*".
const (
	languageCodeRe = `([a-z]{2,3}|\*)`
	scriptCodeRe   = `(?:-([a-z]{4}|\*))`
	regionCodeRe   = `(?:-([a-z]{2}|\*))`
	variantCodeRe  = `(?:-(([0-9][a-z0-9]{3}|[a-z0-9]{5,8})|\*))`
)

// localeRe splits a locale id into four pieces (language, script, region,
// variant), accepting a "*" range char in any position. Matching is
// case-insensitive; the input is lower-cased and underscores normalised to
// dashes before matching, mirroring locale.ts.
var localeRe = regexp.MustCompile(
	`(?i)^` + languageCodeRe + scriptCodeRe + `?` + regionCodeRe + `?` + variantCodeRe + `?$`,
)

// Locale is a parsed BCP-47 locale id. An empty (unset) field is represented by
// the empty string. The Wellformed flag reports whether the id parsed at all.
// Mirrors the fluent.js Locale class; in fluent.js absent fields are
// `undefined`, here they are "".
type Locale struct {
	Wellformed bool
	Language   string
	Script     string
	Region     string
	Variant    string
}

// NewLocale parses a locale id. Underscores are normalised to dashes; the
// language is lower-cased, the script title-cased and the region upper-cased,
// matching locale.ts.
func NewLocale(locale string) *Locale {
	l := &Locale{}
	result := localeRe.FindStringSubmatch(strings.ReplaceAll(locale, "_", "-"))
	if result == nil {
		l.Wellformed = false
		return l
	}

	// result[1]=language, result[2]=script, result[3]=region, result[4]=variant.
	language, script, region, variant := result[1], result[2], result[3], result[4]

	if language != "" {
		l.Language = strings.ToLower(language)
	}
	if script != "" {
		l.Script = strings.ToUpper(script[:1]) + strings.ToLower(script[1:])
	}
	if region != "" {
		l.Region = strings.ToUpper(region)
	}
	l.Variant = strings.ToLower(variant)
	l.Wellformed = true
	return l
}

// IsEqual reports whether two locales have identical fields.
func (l *Locale) IsEqual(other *Locale) bool {
	return l.Language == other.Language &&
		l.Script == other.Script &&
		l.Region == other.Region &&
		l.Variant == other.Variant
}

// Matches reports whether the receiver matches other field-by-field. When
// thisRange is true an empty field on the receiver acts as a wildcard; when
// otherRange is true an empty field on other acts as a wildcard. Mirrors
// Locale.matches in locale.ts (where undefined fields are the empty string
// here).
func (l *Locale) Matches(other *Locale, thisRange, otherRange bool) bool {
	return fieldMatches(l.Language, other.Language, thisRange, otherRange) &&
		fieldMatches(l.Script, other.Script, thisRange, otherRange) &&
		fieldMatches(l.Region, other.Region, thisRange, otherRange) &&
		fieldMatches(l.Variant, other.Variant, thisRange, otherRange)
}

func fieldMatches(a, b string, aRange, bRange bool) bool {
	return a == b || (aRange && a == "") || (bRange && b == "")
}

// String renders the locale id by joining the non-empty fields with dashes,
// mirroring Locale.toString.
func (l *Locale) String() string {
	parts := make([]string, 0, 4)
	for _, p := range []string{l.Language, l.Script, l.Region, l.Variant} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "-")
}

// ClearVariants clears the variant field.
func (l *Locale) ClearVariants() { l.Variant = "" }

// ClearRegion clears the region field.
func (l *Locale) ClearRegion() { l.Region = "" }

// AddLikelySubtags expands the locale using the curated likely-subtags table.
// It returns true and mutates the receiver if an expansion was found, mirroring
// Locale.addLikelySubtags.
func (l *Locale) AddLikelySubtags() bool {
	newLocale := getLikelySubtagsMin(strings.ToLower(l.String()))
	if newLocale != nil {
		l.Language = newLocale.Language
		l.Script = newLocale.Script
		l.Region = newLocale.Region
		l.Variant = newLocale.Variant
		return true
	}
	return false
}

// likelySubtagsMin is the curated subset of the Unicode CLDR likelySubtags
// list, ported verbatim from @fluent/langneg (locale.ts).
var likelySubtagsMin = map[string]string{
	"ar":      "ar-arab-eg",
	"az-arab": "az-arab-ir",
	"az-ir":   "az-arab-ir",
	"be":      "be-cyrl-by",
	"da":      "da-latn-dk",
	"el":      "el-grek-gr",
	"en":      "en-latn-us",
	"fa":      "fa-arab-ir",
	"ja":      "ja-jpan-jp",
	"ko":      "ko-kore-kr",
	"pt":      "pt-latn-br",
	"sr":      "sr-cyrl-rs",
	"sr-ru":   "sr-latn-ru",
	"sv":      "sv-latn-se",
	"ta":      "ta-taml-in",
	"uk":      "uk-cyrl-ua",
	"zh":      "zh-hans-cn",
	"zh-hant": "zh-hant-tw",
	"zh-hk":   "zh-hant-hk",
	"zh-mo":   "zh-hant-mo",
	"zh-tw":   "zh-hant-tw",
	"zh-gb":   "zh-hant-gb",
	"zh-us":   "zh-hant-us",
}

// regionMatchingLangs is the set of languages for which the likely region is
// simply the upper-cased language code (e.g. fr -> fr-FR). Ported from
// locale.ts.
var regionMatchingLangs = map[string]bool{
	"az": true, "bg": true, "cs": true, "de": true, "es": true,
	"fi": true, "fr": true, "hu": true, "it": true, "lt": true,
	"lv": true, "nl": true, "pl": true, "ro": true, "ru": true,
}

// getLikelySubtagsMin returns the maximised Locale for the given lower-cased
// locale id, or nil if no expansion is known. Mirrors getLikelySubtagsMin.
func getLikelySubtagsMin(loc string) *Locale {
	if mapped, ok := likelySubtagsMin[loc]; ok {
		return NewLocale(mapped)
	}
	locale := NewLocale(loc)
	if locale.Language != "" && regionMatchingLangs[locale.Language] {
		locale.Region = strings.ToUpper(locale.Language)
		return locale
	}
	return nil
}
