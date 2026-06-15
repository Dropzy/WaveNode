package database

import "testing"

func TestCalculateScanProgress(t *testing.T) {
	tests := []struct {
		name      string
		processed int
		total     int
		expected  int
	}{
		{name: "no total", processed: 5, total: 0, expected: 0},
		{name: "no progress", processed: 0, total: 52, expected: 0},
		{name: "partial progress rounds", processed: 27, total: 52, expected: 52},
		{name: "complete", processed: 52, total: 52, expected: 100},
		{name: "clamped", processed: 60, total: 52, expected: 100},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := calculateScanProgress(test.processed, test.total); actual != test.expected {
				t.Fatalf("calculateScanProgress(%d, %d) = %d, expected %d", test.processed, test.total, actual, test.expected)
			}
		})
	}
}
