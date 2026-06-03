package plural

import "testing"

func TestOperandsFromString(t *testing.T) {
	cases := []struct {
		in   string
		want Operands
	}{
		{"1", Operands{N: 1, I: 1, V: 0, W: 0, F: 0, T: 0, C: 0}},
		{"1.0", Operands{N: 1, I: 1, V: 1, W: 0, F: 0, T: 0, C: 0}},
		{"1.50", Operands{N: 1.5, I: 1, V: 2, W: 1, F: 50, T: 5, C: 0}},
		{"1.230", Operands{N: 1.23, I: 1, V: 3, W: 2, F: 230, T: 23, C: 0}},
		{"0", Operands{N: 0, I: 0}},
		{"-7.5", Operands{N: 7.5, I: 7, V: 1, W: 1, F: 5, T: 5}},
		{"1000000", Operands{N: 1000000, I: 1000000}},
		// compact exponent scales the value; c retains the exponent.
		{"1c6", Operands{N: 1000000, I: 1000000, C: 6}},
		{"1.0000001c6", Operands{N: 1000000.1, I: 1000000, V: 1, W: 1, F: 1, T: 1, C: 6}},
	}
	for _, c := range cases {
		got, err := OperandsFromString(c.in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q: got %+v want %+v", c.in, got, c.want)
		}
	}
}

func TestNewOperands(t *testing.T) {
	cases := []struct {
		n            float64
		minF, maxF   int
		wantV, wantW int
		wantI        int64
		wantF, wantT int64
	}{
		{1, 0, 0, 0, 0, 1, 0, 0},
		{1, 1, 3, 1, 0, 1, 0, 0},   // -> "1.0"
		{1.5, 0, 3, 1, 1, 1, 5, 5}, // -> "1.5"
		{1.5, 2, 2, 2, 1, 1, 50, 5},
		{2.0, 1, 1, 1, 0, 2, 0, 0},
	}
	for _, c := range cases {
		o := NewOperands(c.n, c.minF, c.maxF)
		if o.V != c.wantV || o.W != c.wantW || o.I != c.wantI || o.F != c.wantF || o.T != c.wantT {
			t.Errorf("NewOperands(%v,%d,%d) = %+v", c.n, c.minF, c.maxF, o)
		}
	}
}

func TestLocaleFallback(t *testing.T) {
	// en-US should fall back to en (one/other).
	if got := CardinalFor("en-US", 1, 0, 0); got != One {
		t.Errorf("en-US 1 = %q want one", got)
	}
	if got := CardinalFor("en-US", 2, 0, 0); got != Other {
		t.Errorf("en-US 2 = %q want other", got)
	}
	// pt-PT is region-specific: 1 -> one (i=1,v=0), but 0 -> other (unlike pt).
	if got := CardinalFor("pt-PT", 1, 0, 0); got != One {
		t.Errorf("pt-PT 1 = %q want one", got)
	}
	if got := CardinalFor("pt-PT", 0, 0, 0); got != Other {
		t.Errorf("pt-PT 0 = %q want other", got)
	}
	// pt: 0 and 1 are both one.
	if got := CardinalFor("pt", 0, 0, 0); got != One {
		t.Errorf("pt 0 = %q want one", got)
	}
	// unknown locale -> root/other.
	if got := CardinalFor("zz", 5, 0, 0); got != Other {
		t.Errorf("zz 5 = %q want other", got)
	}
	// underscore form normalises.
	if got := CardinalFor("pt_PT", 0, 0, 0); got != Other {
		t.Errorf("pt_PT 0 = %q want other", got)
	}
}

func TestRussianCardinal(t *testing.T) {
	cases := []struct {
		n    int64
		want Category
	}{
		{1, One}, {21, One}, {2, Few}, {3, Few}, {4, Few},
		{5, Many}, {11, Many}, {12, Many}, {0, Many}, {100, Many},
	}
	for _, c := range cases {
		if got := CardinalFor("ru", float64(c.n), 0, 0); got != c.want {
			t.Errorf("ru %d = %q want %q", c.n, got, c.want)
		}
	}
}
