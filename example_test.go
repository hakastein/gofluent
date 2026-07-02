package fluent_test

import (
	"fmt"
	"time"

	_ "github.com/hakastein/gocldr/locales/ru" // opt-in CLDR data: Russian numbers + dates

	fluent "github.com/hakastein/gofluent"
)

// Example shows the minimal flow: parse a resource, add it to a bundle, look up
// a message, and format its pattern with arguments.
func Example() {
	res := fluent.NewResource("hello = Hello, { $name }!")

	// useIsolating is disabled here so the output is plain ASCII; in production
	// the default (true) wraps placeables in Unicode bidi isolation marks.
	b := fluent.NewBundle("en", fluent.WithUseIsolating(false))
	b.AddResource(res)

	msg, ok := b.Message("hello")
	if !ok {
		panic("message not found")
	}

	out, _ := b.FormatPattern(msg.Value(), map[string]any{"name": "World"})
	fmt.Println(out)
	// Output: Hello, World!
}

// ExampleBundle_selectExpression demonstrates a select expression. A numeric
// selector matches a variant key by exact value before falling back to the CLDR
// plural category, so the literal key "1" is selected for the number 1 while 5
// falls through to *[other].
func ExampleBundle_selectExpression() {
	src := `
emails =
    { $count ->
        [1] You have one new email.
       *[other] You have { $count } new emails.
    }
`
	b := fluent.NewBundle("en", fluent.WithUseIsolating(false))
	b.AddResource(fluent.NewResource(src))

	msg, _ := b.Message("emails")
	one, _ := b.FormatPattern(msg.Value(), map[string]any{"count": 1})
	five, _ := b.FormatPattern(msg.Value(), map[string]any{"count": 5})
	fmt.Println(one)
	fmt.Println(five)
	// Output:
	// You have one new email.
	// You have 5 new emails.
}

// ExampleBundle_pluralRussian shows a Russian bundle: the { $n -> [one] … [few] …
// *[many] … } select picks the correct CLDR plural category, and NUMBER()/
// DATETIME() render with Russian conventions. CLDR formatting is the NewBundle
// default, so no extra wiring is needed.
//
// The blank import _ "github.com/hakastein/gocldr/locales/ru" supplies Russian
// number and date data; CLDR plural rules are always linked, so the category
// selection (one/few/many) is correct even without it.
func ExampleBundle_pluralRussian() {
	const src = `
apples =
    { $n ->
        [one] { $n } яблоко
        [few] { $n } яблока
       *[many] { $n } яблок
    }
total = Итого: { NUMBER($total) }
updated = Обновлено { DATETIME($at, dateStyle: "long") }
`
	// CLDR formatters are installed by default; useIsolating is disabled so the
	// output is plain text.
	b := fluent.NewBundle("ru", fluent.WithUseIsolating(false))
	b.AddResource(fluent.NewResource(src))

	apples, _ := b.Message("apples")
	for _, n := range []int{1, 2, 5, 21} {
		out, _ := b.FormatPattern(apples.Value(), map[string]any{"n": n})
		fmt.Println(out)
	}

	total, _ := b.Message("total")
	out, _ := b.FormatPattern(total.Value(), map[string]any{"total": 1234567})
	fmt.Println(out)

	updated, _ := b.Message("updated")
	at := time.Date(2023, 1, 5, 14, 9, 7, 0, time.UTC)
	out, _ = b.FormatPattern(updated.Value(), map[string]any{"at": at})
	fmt.Println(out)

	// Output:
	// 1 яблоко
	// 2 яблока
	// 5 яблок
	// 21 яблоко
	// Итого: 1 234 567
	// Обновлено 5 января 2023 г.
}
