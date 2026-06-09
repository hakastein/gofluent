package fluent_test

import (
	"fmt"
	"time"

	_ "github.com/hakastein/gocldr/locales/ru" // opt-in CLDR data: Russian numbers + dates

	fluent "github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/fluentx"
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

// ExampleBundle_pluralRussian wires the fluentx (CLDR) formatters into a Russian
// bundle so the { $n -> [one] … [few] … *[many] … } select picks the correct
// CLDR plural category, and NUMBER()/DATETIME() render with Russian conventions.
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
	res, errs := fluent.NewResource(src)
	if len(errs) > 0 {
		panic(errs[0])
	}

	// fluentx.Options() injects the CLDR plural rules, number, and datetime
	// formatters. useIsolating is disabled so the output is plain text.
	b := fluent.NewBundle("ru", append(fluentx.Options(), fluent.WithUseIsolating(false))...)
	b.AddResource(res)

	apples, _ := b.GetMessage("apples")
	for _, n := range []int{1, 2, 5, 21} {
		fmt.Println(b.FormatPatternAny(apples.Value, map[string]any{"n": n}, nil))
	}

	total, _ := b.GetMessage("total")
	fmt.Println(b.FormatPatternAny(total.Value, map[string]any{"total": 1234567}, nil))

	updated, _ := b.GetMessage("updated")
	at := time.Date(2023, 1, 5, 14, 9, 7, 0, time.UTC)
	fmt.Println(b.FormatPatternAny(updated.Value, map[string]any{"at": at}, nil))

	// Output:
	// 1 яблоко
	// 2 яблока
	// 5 яблок
	// 21 яблоко
	// Итого: 1 234 567
	// Обновлено 5 января 2023 г.
}
