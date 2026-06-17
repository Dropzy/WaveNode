package artistmeta

import "testing"

func TestIsReusableLicense(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "CC BY-SA 4.0", want: true},
		{name: "cc-by-3.0", want: true},
		{name: "CC0", want: true},
		{name: "Public domain", want: true},
		{name: "All rights reserved", want: false},
		{name: "fair use", want: false},
		{name: "", want: false},
	}

	for _, test := range tests {
		if got := IsReusableLicense(test.name); got != test.want {
			t.Fatalf("IsReusableLicense(%q) = %v, want %v", test.name, got, test.want)
		}
	}
}

func TestBuildAttribution(t *testing.T) {
	got := BuildAttribution("Jane Photographer", "CC BY-SA 4.0", "https://commons.wikimedia.org/wiki/File:artist.jpg")
	want := "Jane Photographer · CC BY-SA 4.0 · https://commons.wikimedia.org/wiki/File:artist.jpg"
	if got != want {
		t.Fatalf("BuildAttribution() = %q, want %q", got, want)
	}

	publicDomain := BuildAttribution("", "Public domain", "https://commons.wikimedia.org/wiki/File:artist.jpg")
	if publicDomain != "Public domain image via https://commons.wikimedia.org/wiki/File:artist.jpg" {
		t.Fatalf("public domain attribution = %q", publicDomain)
	}
}
