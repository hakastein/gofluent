package fluent_test

import (
	"sync"
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
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
			b.FormatPattern(msg.Value, nil)
			b.Message("greet")
		}
	}()

	wg.Wait()
}
