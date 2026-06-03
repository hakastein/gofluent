// Package langneg is a faithful Go port of @fluent/langneg. It negotiates an
// ordered list of supported locales out of a user's requested locales and an
// application's available locales, using the Fluent variant of the BCP-4647
// Extended Filtering algorithm plus a curated likely-subtags table.
//
// The port is dependency-free (it does not use golang.org/x/text); it carries
// its own minimal likely-subtags data, matching the historical self-contained
// @fluent/langneg implementation.
package langneg

import (
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Strategy selects the negotiation algorithm. Mirrors the
// "filtering" | "matching" | "lookup" option in fluent.js.
type Strategy int

const (
	// Filtering (the default) matches as many available locales as possible for
	// each requested locale, in requested-locale priority order.
	Filtering Strategy = iota
	// Matching returns the single best available locale for each requested
	// locale.
	Matching
	// Lookup returns a single best available locale for the whole request and
	// requires a non-empty defaultLocale as the last resort.
	Lookup
)

// String renders the strategy using the fluent.js option names.
func (s Strategy) String() string {
	switch s {
	case Matching:
		return "matching"
	case Lookup:
		return "lookup"
	default:
		return "filtering"
	}
}

// ErrLookupNeedsDefault is returned by NegotiateLanguagesErr when the Lookup
// strategy is used without a defaultLocale, mirroring the thrown Error in
// fluent.js.
var ErrLookupNeedsDefault = errors.New("langneg: defaultLocale cannot be empty for strategy lookup")

// NegotiateLanguages negotiates requested against available locales, appending
// defaultLocale as a last resort, and returns the supported locales in priority
// order. Mirrors negotiateLanguages in fluent.js.
//
// For the Lookup strategy a non-empty defaultLocale is required; if it is empty
// this function panics (matching fluent.js, which throws). Use
// NegotiateLanguagesErr for an error-returning variant.
func NegotiateLanguages(requested, available []string, defaultLocale string, strategy Strategy) []string {
	result, err := NegotiateLanguagesErr(requested, available, defaultLocale, strategy)
	if err != nil {
		panic(err)
	}
	return result
}

// NegotiateLanguagesErr is the error-returning variant of NegotiateLanguages.
// It returns ErrLookupNeedsDefault when the Lookup strategy is used with an
// empty defaultLocale instead of panicking.
func NegotiateLanguagesErr(requested, available []string, defaultLocale string, strategy Strategy) ([]string, error) {
	supported := FilterMatches(requested, available, strategy)

	if strategy == Lookup {
		if defaultLocale == "" {
			return nil, ErrLookupNeedsDefault
		}
		if len(supported) == 0 {
			supported = append(supported, defaultLocale)
		}
	} else if defaultLocale != "" && !contains(supported, defaultLocale) {
		supported = append(supported, defaultLocale)
	}

	return supported, nil
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// acceptedEntryRe matches one Accept-Language entry plus its optional q value:
//
//	(?:^|,)([^,;]+)(?:;\s*[qQ]\s*=([^,;]+))?
var acceptedEntryRe = regexp.MustCompile(`(?:^|,)([^,;]+)(?:;\s*[qQ]\s*=([^,;]+))?`)

// AcceptedLanguages parses an HTTP Accept-Language header value into an ordered
// list of locale ids, sorted by descending q value with header order preserved
// for equal weights. Mirrors acceptedLanguages in fluent.js (the current,
// q-aware implementation). An empty header yields nil.
func AcceptedLanguages(header string) []string {
	var entries []langQ
	for _, m := range acceptedEntryRe.FindAllStringSubmatchIndex(header, -1) {
		// m[2]:m[3] is the lang group, m[4]:m[5] is the q value group.
		lang := strings.TrimSpace(header[m[2]:m[3]])
		q := 1.0
		if m[4] >= 0 {
			q = parseQ(header[m[4]:m[5]])
		}
		entries = append(entries, langQ{lang: lang, q: q, index: m[0]})
	}

	// Stable sort by q descending; equal weights keep header (index) order.
	stableSortByQ(entries)

	if len(entries) == 0 {
		return nil
	}
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.lang
	}
	return out
}

// leadingFloatRe captures the leading numeric prefix of a string, mirroring
// JavaScript's parseFloat which parses as much of a number as it can from the
// start and ignores trailing garbage.
var leadingFloatRe = regexp.MustCompile(`^\s*[-+]?(\d+\.?\d*|\.\d+)([eE][-+]?\d+)?`)

// parseQ mirrors `parseFloat(token) || 0`: it parses the leading numeric prefix
// of token and returns 0 when no number is present (NaN -> 0).
func parseQ(token string) float64 {
	m := leadingFloatRe.FindString(token)
	if m == "" {
		return 0
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(m), 64)
	if err != nil {
		return 0
	}
	return v
}

// langQ is one parsed Accept-Language entry: its locale id, q weight, and the
// byte offset at which it appeared (used as a stable tiebreaker).
type langQ struct {
	lang  string
	q     float64
	index int
}

// stableSortByQ sorts entries by descending q, keeping header order (index) for
// equal weights. Mirrors the comparator (a.q === b.q ? a.index - b.index : b.q - a.q).
func stableSortByQ(entries []langQ) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].q == entries[j].q {
			return entries[i].index < entries[j].index
		}
		return entries[i].q > entries[j].q
	})
}
