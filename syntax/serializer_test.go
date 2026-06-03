package syntax

import "testing"

func TestSerializeRoundTrip(t *testing.T) {
	cases := []string{
		"foo = Bar\n",
		"-brand = Firefox\n",
		"foo = Bar\n    .attr = Baz\n",
		"foo =\n    Multi\n    line\n",
		"foo = { $n ->\n        [one] One\n       *[other] Other\n    }\n",
		"foo = { FUN($a, key: 1) }\n",
		"foo = Pre { $var } post\n",
	}
	for _, src := range cases {
		res := Parse(src)
		got := Serialize(res)
		// Re-parse and re-serialize to confirm a stable canonical form.
		got2 := Serialize(Parse(got))
		if got != got2 {
			t.Errorf("not idempotent for %q:\n first: %q\nsecond: %q", src, got, got2)
		}
	}
}

func TestSerializeStableForCanonicalInput(t *testing.T) {
	// These inputs are already in canonical form; serialize must reproduce them.
	cases := []string{
		"foo = Bar\n",
		"-brand = Firefox\n",
		"foo = Bar\n    .attr = Baz\n",
	}
	for _, src := range cases {
		if got := Serialize(Parse(src)); got != src {
			t.Errorf("Serialize(Parse(%q)) = %q, want identity", src, got)
		}
	}
}

func TestSerializeWithJunk(t *testing.T) {
	src := "err = {1xx}\ngood = Value\n"
	res := Parse(src)
	withoutJunk := Serialize(res)
	if withoutJunk != "good = Value\n" {
		t.Errorf("without junk = %q", withoutJunk)
	}
	withJunk := Serialize(res, WithJunk(true))
	if withJunk != src {
		t.Errorf("with junk = %q, want %q", withJunk, src)
	}
}

func TestSerializeComment(t *testing.T) {
	src := "# A comment\nfoo = Bar\n"
	res := Parse(src)
	if got := Serialize(res); got != src {
		t.Errorf("comment serialize = %q, want %q", got, src)
	}
}
