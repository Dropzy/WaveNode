package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestRefreshStatusKeepsBackendVersion(t *testing.T) {
	updater := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"current_version":"0.1.2",
			"latest_version":"v0.1.5",
			"state":"completed",
			"message":"WaveNode containers updated."
		}`))
	}))
	defer updater.Close()

	t.Setenv("WAVENODE_UPDATER_URL", updater.URL)
	manager := NewUpdateManager("v0.1.5")
	status := manager.RefreshStatus(context.Background())

	if status.CurrentVersion != "v0.1.5" {
		t.Fatalf("CurrentVersion = %q, want backend version %q", status.CurrentVersion, "v0.1.5")
	}
	if status.LatestVersion != "v0.1.5" || status.State != "completed" {
		t.Fatalf("updater status was not merged: %#v", status)
	}
}
