package fluent_test

import (
	"fmt"

	fluent "github.com/hakastein/gofluent"
)

// Example shows the minimal flow: parse a resource, add it to a bundle, look up
// a message, and format its pattern with arguments.
func Example() {
	res, errs := fluent.NewResource("hello = Hello, { $name }!")
	if len(errs) > 0 {
		panic(errs[0])
	}

	// useIsolating is disabled here so the output is plain ASCII; in production
	// the default (true) wraps placeables in Unicode bidi isolation marks.
	b := fluent.NewBundle("en", fluent.WithUseIsolating(false))
	b.AddResource(res)

	msg, ok := b.GetMessage("hello")
	if !ok {
		panic("message not found")
	}

	var ferrs []error
	out := b.FormatPatternAny(msg.Value, map[string]any{"name": "World"}, &ferrs)
	fmt.Println(out)
	// Output: Hello, World!
}

// ExampleBundle_selectExpression demonstrates a select expression. Without a
// real PluralRules implementation wired in, numeric selectors match by exact
// value, so the variant key "1" is selected for the number 1.
func ExampleBundle_selectExpression() {
	src := `
emails =
    { $count ->
        [1] You have one new email.
       *[other] You have { $count } new emails.
    }
`
	res, _ := fluent.NewResource(src)
	b := fluent.NewBundle("en", fluent.WithUseIsolating(false))
	b.AddResource(res)

	msg, _ := b.GetMessage("emails")
	var errs []error
	fmt.Println(b.FormatPatternAny(msg.Value, map[string]any{"count": 1}, &errs))
	fmt.Println(b.FormatPatternAny(msg.Value, map[string]any{"count": 5}, &errs))
	// Output:
	// You have one new email.
	// You have 5 new emails.
}
