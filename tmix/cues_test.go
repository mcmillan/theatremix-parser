package tmix

import "testing"

func TestCueID(t *testing.T) {
	cases := map[[2]int]string{{0, 0}: "0", {5, 0}: "5", {5, 20}: "5.20", {0, 1}: "0.1", {0, 10}: "0.10", {12, 5}: "12.5"}
	for in, want := range cases {
		if got := cueID(in[0], in[1]); got != want {
			t.Errorf("cueID(%d,%d) = %q want %q", in[0], in[1], got, want)
		}
	}
}
