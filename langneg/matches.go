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

// FilterMatches negotiates requestedLocales against availableLocales using the
// given strategy, returning the supported locales (keys taken from
// availableLocales) in priority order. Mirrors filterMatches in matches.ts.
func FilterMatches(requestedLocales, availableLocales []string, strategy Strategy) []string {
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

	// removeKey deletes a matched available locale so it cannot match twice.
	removeKey := func(key string) {
		for i, entry := range availableMap {
			if entry.key == key {
				availableMap = append(availableMap[:i], availableMap[i+1:]...)
				return
			}
		}
	}

outer:
	for _, reqLocStr := range requestedLocales {
		reqLocStrLC := strings.ToLower(reqLocStr)
		requestedLocale := NewLocale(reqLocStrLC)

		if requestedLocale.Language == "" {
			continue
		}

		// runPass applies pred to every remaining available locale. For
		// filtering it keeps collecting matches; for matching/lookup the first
		// match advances/terminates. It returns (advance, stop) where advance
		// means "continue outer" and stop means "return immediately" (lookup).
		runPass := func(pred func(available *Locale) bool) (advance bool, stop bool) {
			// Iterate over a snapshot of keys so deletions are safe.
			keys := make([]string, len(availableMap))
			locales := make([]*Locale, len(availableMap))
			for i, entry := range availableMap {
				keys[i] = entry.key
				locales[i] = entry.locale
			}
			for i, key := range keys {
				if !supportedSet[key] && pred(locales[i]) {
					addSupported(key)
					removeKey(key)
					switch strategy {
					case Lookup:
						return false, true
					case Filtering:
						continue
					default: // Matching
						return true, false
					}
				}
			}
			return false, false
		}

		// 1) Exact match. Example: `en-US` === `en-US`.
		{
			keys := make([]string, len(availableMap))
			for i, entry := range availableMap {
				keys[i] = entry.key
			}
			for _, key := range keys {
				if supportedSet[key] {
					continue
				}
				if reqLocStrLC == strings.ToLower(key) {
					addSupported(key)
					removeKey(key)
					switch strategy {
					case Lookup:
						return supported
					case Filtering:
						continue
					default:
						continue outer
					}
				}
			}
		}

		// 2) Match against the available range (available treated as wildcard).
		// Example: ['en-US'] * ['en'] = ['en'].
		if advance, stop := runPass(func(available *Locale) bool {
			return available.Matches(requestedLocale, true, false)
		}); stop {
			return supported
		} else if advance {
			continue outer
		}

		// 3) Maximal (likely-subtags) version of the request.
		// Example: ['en'] * ['en-GB','en-US'] = ['en-US'].
		if requestedLocale.AddLikelySubtags() {
			if advance, stop := runPass(func(available *Locale) bool {
				return available.Matches(requestedLocale, true, false)
			}); stop {
				return supported
			} else if advance {
				continue outer
			}
		}

		// 4) Different variant for the same locale id.
		// Example: ['en-US-mac'] * ['en-US-win'] = ['en-US-win'].
		requestedLocale.ClearVariants()
		if advance, stop := runPass(func(available *Locale) bool {
			return available.Matches(requestedLocale, true, true)
		}); stop {
			return supported
		} else if advance {
			continue outer
		}

		// 5) Likely subtag without region.
		// Example: ['zh-Hant-HK'] * ['zh-TW','zh-CN'] = ['zh-TW'].
		requestedLocale.ClearRegion()
		if requestedLocale.AddLikelySubtags() {
			if advance, stop := runPass(func(available *Locale) bool {
				return available.Matches(requestedLocale, true, false)
			}); stop {
				return supported
			} else if advance {
				continue outer
			}
		}

		// 6) Different region for the same locale id.
		// Example: ['en-US'] * ['en-AU'] = ['en-AU'].
		requestedLocale.ClearRegion()
		if _, stop := runPass(func(available *Locale) bool {
			return available.Matches(requestedLocale, true, true)
		}); stop {
			return supported
		}
	}

	return supported
}
