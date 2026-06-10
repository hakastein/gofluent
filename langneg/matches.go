package langneg

import "strings"

// The negotiation algorithm is based on BCP-4647 3.3.2 Extended Filtering with
// three Fluent-specific modifications:
//
//  1. Available locales are treated as ranges, so a more specific request can
//     match a more generic available locale (['en-US'] * ['en'] = ['en']).
//  2. Likely subtags from LDML 4.3 are applied via the curated table in
//     locale.go (['fr'] * ['fr-FR'] = ['fr-FR']).
//  3. A variant/region range check lets a request fall back to a different
//     region/variant of the same language/script before falling back to the
//     next requested language.

// orderedLocale pairs an available locale's original key string with its parsed
// form, preserving insertion (available-list) order — Go maps do not, and the
// JS Map the port mirrors does.
type orderedLocale struct {
	key    string
	locale *Locale
}

// filterMatches negotiates requestedLocales against availableLocales using the
// given strategy, returning the supported locales (keys taken from
// availableLocales) in priority order. Mirrors filterMatches in matches.ts.
func filterMatches(requestedLocales, availableLocales []string, strategy Strategy) []string {
	var supported []string
	supportedSet := make(map[string]bool)

	addSupported := func(key string) {
		if !supportedSet[key] {
			supportedSet[key] = true
			supported = append(supported, key)
		}
	}

	// Parse the available locales, preserving order and dropping malformed ids.
	availableMap := make([]orderedLocale, 0, len(availableLocales))
	for _, locale := range availableLocales {
		if parsed := NewLocale(locale); parsed.Wellformed {
			availableMap = append(availableMap, orderedLocale{key: locale, locale: parsed})
		}
	}

outer:
	for _, reqLocStr := range requestedLocales {
		reqLocStrLC := strings.ToLower(reqLocStr)
		requestedLocale := NewLocale(reqLocStrLC)

		if requestedLocale.Language == "" {
			continue
		}

		// runPass applies pred to every remaining available locale, consuming
		// matches so they cannot match twice. For Filtering it keeps collecting
		// matches; for Matching/Lookup the first match ends the pass. It returns
		// (advance, stop) where advance means "continue with the next requested
		// locale" and stop means "negotiation is complete" (Lookup).
		runPass := func(pred func(entry orderedLocale) bool) (advance, stop bool) {
			matched := false
			kept := availableMap[:0]
			for _, entry := range availableMap {
				if (strategy == Filtering || !matched) && !supportedSet[entry.key] && pred(entry) {
					addSupported(entry.key)
					matched = true
					continue
				}
				kept = append(kept, entry)
			}
			availableMap = kept
			if !matched {
				return false, false
			}
			switch strategy {
			case Lookup:
				return false, true
			case Filtering:
				return false, false
			default: // Matching
				return true, false
			}
		}

		// 1) Exact match. Example: `en-US` === `en-US`.
		if advance, stop := runPass(func(e orderedLocale) bool {
			return reqLocStrLC == strings.ToLower(e.key)
		}); stop {
			return supported
		} else if advance {
			continue outer
		}

		// 2) Match against the available range (available treated as wildcard).
		// Example: ['en-US'] * ['en'] = ['en'].
		if advance, stop := runPass(func(e orderedLocale) bool {
			return e.locale.matches(requestedLocale, true, false)
		}); stop {
			return supported
		} else if advance {
			continue outer
		}

		// 3) Maximal (likely-subtags) version of the request.
		// Example: ['en'] * ['en-GB','en-US'] = ['en-US'].
		if requestedLocale.addLikelySubtags() {
			if advance, stop := runPass(func(e orderedLocale) bool {
				return e.locale.matches(requestedLocale, true, false)
			}); stop {
				return supported
			} else if advance {
				continue outer
			}
		}

		// 4) Different variant for the same locale id.
		// Example: ['en-US-mac'] * ['en-US-win'] = ['en-US-win'].
		requestedLocale.clearVariant()
		if advance, stop := runPass(func(e orderedLocale) bool {
			return e.locale.matches(requestedLocale, true, true)
		}); stop {
			return supported
		} else if advance {
			continue outer
		}

		// 5) Likely subtag without region.
		// Example: ['zh-Hant-HK'] * ['zh-TW','zh-CN'] = ['zh-TW'].
		requestedLocale.clearRegion()
		if requestedLocale.addLikelySubtags() {
			if advance, stop := runPass(func(e orderedLocale) bool {
				return e.locale.matches(requestedLocale, true, false)
			}); stop {
				return supported
			} else if advance {
				continue outer
			}
		}

		// 6) Different region for the same locale id.
		// Example: ['en-US'] * ['en-AU'] = ['en-AU'].
		requestedLocale.clearRegion()
		if _, stop := runPass(func(e orderedLocale) bool {
			return e.locale.matches(requestedLocale, true, true)
		}); stop {
			return supported
		}
	}

	return supported
}
