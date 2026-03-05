package workspaceid

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty uses canonical default", raw: "", want: CanonicalDefault},
		{name: "whitespace uses canonical default", raw: "   ", want: CanonicalDefault},
		{name: "default is explicit workspace", raw: "default", want: "default"},
		{name: "mixed case default is explicit workspace", raw: "DeFaUlT", want: "DeFaUlT"},
		{name: "explicit workspace preserved", raw: "ws-explicit", want: "ws-explicit"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Normalize(tc.raw); got != tc.want {
				t.Fatalf("Normalize(%q)=%q want %q", tc.raw, got, tc.want)
			}
		})
	}
}
