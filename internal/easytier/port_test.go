package easytier

import "testing"

func TestParsePortFromAddress(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{name: "host_port", input: "127.0.0.1:15888", want: 15888},
		{name: "ipv6_port", input: "[::1]:11010", want: 11010},
		{name: "port_only", input: "11011", want: 11011},
		{name: "invalid_empty", input: "", want: 0},
		{name: "invalid_text", input: "abc", want: 0},
		{name: "invalid_range", input: "70000", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParsePortFromAddress(tc.input); got != tc.want {
				t.Fatalf("ParsePortFromAddress(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}
