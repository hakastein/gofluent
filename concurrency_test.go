package fluent_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/hakastein/gocldr/locales/all" // CLDR data so the default formatters render real locale tables

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBundleConcurrentAccess exercises Bundle for the data race documented in
// the contract: AddFunction/AddResource/AddResourceOverriding mutate the
// bundle's maps while FormatPattern/Message read them. Run with -race to catch
// the regression.
func TestBundleConcurrentAccess(t *testing.T) {
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
	b.AddResource(fluent.NewResource("greet = { ECHO() } world\n"))

	echo := func(_ []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		return fluent.String("hello"), nil
	}
	b.AddFunction("ECHO", echo)

	msg, ok := b.Message("greet")
	assert.True(t, ok)

	var wg sync.WaitGroup

	// Writer goroutine: continuously mutate the maps through every Add method.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			b.AddFunction("ECHO", echo)
			b.AddResource(fluent.NewResource("extra = Extra\n"))
			b.AddResourceOverriding(fluent.NewResource("extra = Extra\n"))
		}
	}()

	// Reader goroutine: format a pattern that calls a function and reference the
	// bundle's messages map.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			b.FormatPattern(msg.Value(), nil)
			b.Message("greet")
		}
	}()

	wg.Wait()
}

// TestBundleConcurrentDefaultFormatters runs the CLDR-backed formatters that
// NewBundle installs by default under concurrency. The trivial-function test
// above cannot surface a data race hiding in lazy, unsynchronized state inside
// the default NUMBER / DATETIME / plural-select path, because it never
// exercises that path. Here multiple readers format patterns that drive number
// formatting (with options), date/time formatting, CLDR plural selection, and
// term references and term attributes, while multiple writers mutate the
// bundle's maps through every Add method. Run with -race.
func TestBundleConcurrentDefaultFormatters(t *testing.T) {
	const src = "-brand = Aurora\n" +
		"    .tagline = shine\n" +
		"count = { NUMBER($n, minimumFractionDigits: 2) } items\n" +
		"when = { DATETIME($d, dateStyle: \"long\") }\n" +
		"apples = { $n ->\n" +
		"        [one] { $n } apple\n" +
		"       *[other] { $n } apples\n" +
		"    }\n" +
		"welcome = Welcome to { -brand }, { -brand.tagline }\n"

	// Default bundle: no custom formatters, isolation on.
	b := fluent.NewBundle("en")
	require.NoError(t, b.AddResource(fluent.NewResource(src)), "AddResource errors")

	ts := time.Date(2023, 1, 5, 14, 9, 7, 0, time.UTC)
	ids := []string{"count", "when", "apples", "welcome"}

	const (
		readers = 6
		writers = 3
		iters   = 200
	)

	var wg sync.WaitGroup
	problems := make(chan error, readers*iters)

	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				id := ids[(seed+i)%len(ids)]
				msg, ok := b.Message(id)
				if !ok || msg.Value() == nil {
					problems <- fmt.Errorf("message %q missing while formatting", id)
					continue
				}
				args := map[string]any{"n": float64((seed + i) % 4), "d": ts}
				out, err := b.FormatPattern(msg.Value(), args)
				if out == "" {
					problems <- fmt.Errorf("empty output for %q", id)
				}
				if err != nil {
					problems <- fmt.Errorf("format %q: %w", id, err)
				}
			}
		}(r)
	}

	// A resource of new ids: the first insert wins, later ones report override
	// errors that are irrelevant here; the point is the concurrent map write.
	extra := fluent.NewResource("solo = Solo\n")
	noop := func(_ []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		return fluent.String(""), nil
	}
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				b.AddResourceOverriding(fluent.NewResource(src))
				b.AddFunction("NOOP", noop)
				b.AddResource(extra)
			}
		}()
	}

	wg.Wait()
	close(problems)
	for err := range problems {
		t.Error(err)
	}
}
