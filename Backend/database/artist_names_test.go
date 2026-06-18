package database

import (
	"reflect"
	"testing"
)

func TestSplitArtistNames(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "comma separated artists",
			in:   "Artist One, Artist Two",
			want: []string{"Artist One", "Artist Two"},
		},
		{
			name: "mixed common separators",
			in:   "Artist One, Artist Two + Artist Three",
			want: []string{"Artist One", "Artist Two", "Artist Three"},
		},
		{
			name: "deduplicates names",
			in:   "Artist One, artist one",
			want: []string{"Artist One"},
		},
		{
			name: "does not split slash names",
			in:   "AC/DC",
			want: []string{"AC/DC"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitArtistNames(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SplitArtistNames(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestPrimaryArtistName(t *testing.T) {
	got := PrimaryArtistName("Artist One, Artist Two")
	if got != "Artist One" {
		t.Fatalf("PrimaryArtistName() = %q, want %q", got, "Artist One")
	}
}

func TestPrimaryArtistNameMatches(t *testing.T) {
	if !PrimaryArtistNameMatches("Artist One, Artist Two", "Artist One") {
		t.Fatal("expected first artist to match")
	}

	if PrimaryArtistNameMatches("Artist One, Artist Two", "Artist Two") {
		t.Fatal("did not expect secondary artist to match")
	}
}
