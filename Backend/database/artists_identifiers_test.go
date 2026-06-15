package database

import "testing"

func TestNullableArtistIdentifier(t *testing.T) {
	if value := nullableArtistIdentifier("  "); value != nil {
		t.Fatalf("blank identifier = %#v, want nil", value)
	}
	if value := nullableArtistIdentifier(" artist-id "); value != "artist-id" {
		t.Fatalf("identifier = %#v, want trimmed value", value)
	}
}
