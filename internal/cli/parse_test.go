package cli

import "testing"

func TestLooksLikeURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  bool
	}{
		{input: "https://medium.com/article", want: true},
		{input: "http://example.com/a/b", want: true},
		{input: "medium.com/article", want: true},
		{input: "example.com", want: true},
		{input: "localhost:8080/path", want: true},
		{input: "search query", want: false},
		{input: "systemd sandboxing", want: false},
		{input: "systemd", want: false},
		{input: "", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := looksLikeURL(tc.input)
			if got != tc.want {
				t.Fatalf("looksLikeURL(%q) = %t, want %t", tc.input, got, tc.want)
			}
		})
	}
}
