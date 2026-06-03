package fluent

// This file ports fluent.js/fluent-bundle/src/memoizer.ts.
//
// In fluent.js the memoizer caches `Intl.*` objects per locale, because
// constructing them is expensive. In this Go port the formatters are injected
// (NumberFormatter, DateTimeFormatter, PluralRules) and are free to do their
// own caching, so there is no Intl object to memoize. The memoizer is therefore
// folded away to a thin, dependency-free helper that preserves the per-locale
// caching shape in case an injected formatter wants to share state keyed by the
// bundle's locale list.

import "sync"

// memoizerCache maps a joined-locale key to an arbitrary per-locale cache slot.
// It mirrors the module-level cache in memoizer.ts. It is unused by the core
// (which delegates all formatting), but exported helpers let an embedder reuse
// the same per-locale memoization strategy.
var (
	memoizerMu    sync.Mutex
	memoizerCache = map[string]map[string]any{}
)

// MemoizerForLocales returns a process-wide cache map shared by all bundles
// using the same locale list. Mirrors getMemoizerForLocale in fluent.js.
func MemoizerForLocales(locales []string) map[string]any {
	key := localesKey(locales)
	memoizerMu.Lock()
	defer memoizerMu.Unlock()
	cache, ok := memoizerCache[key]
	if !ok {
		cache = map[string]any{}
		memoizerCache[key] = cache
	}
	return cache
}
