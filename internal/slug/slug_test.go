package slug

import "testing"

func TestGenerate(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"simple ascii", "buy milk", 40, "buy-milk"},
		{"uppercase and punctuation", "Hello, World!", 40, "hello-world"},
		{"unicode-only yields empty", "🚀", 40, ""},
		{"exactly at cap", "abcd-efgh-ijkl", 14, "abcd-efgh-ijkl"},
		{"just over cap truncates at hyphen boundary", "foo bar baz qux", 10, "foo-bar"},
		{"empty input", "", 40, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Generate(tc.input, tc.maxLen)
			if got != tc.want {
				t.Errorf("Generate(%q, %d) = %q; want %q", tc.input, tc.maxLen, got, tc.want)
			}
		})
	}
}
