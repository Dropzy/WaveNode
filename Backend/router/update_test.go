package router

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "newer patch", left: "v0.1.1", right: "0.1.0", want: 1},
		{name: "same version", left: "v1.2.3", right: "1.2.3", want: 0},
		{name: "older minor", left: "1.1.0", right: "1.2.0", want: -1},
		{name: "missing patch equals zero", left: "1.2", right: "1.2.0", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := compareVersions(test.left, test.right)
			if got != test.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}
