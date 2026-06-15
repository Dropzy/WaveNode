package scanner

import (
	"context"
	"testing"
)

func TestClaimCompletionRejectsPendingStop(t *testing.T) {
	scanner := &Scanner{
		isScanning:  true,
		stopPending: true,
	}

	if scanner.claimCompletion(context.Background()) {
		t.Fatal("completion should not win after a stop request")
	}
	if !scanner.isScanning {
		t.Fatal("scanner should remain active until stopped status is recorded")
	}
}

func TestClaimCompletionClosesCancellationWindow(t *testing.T) {
	scanner := &Scanner{isScanning: true}

	if !scanner.claimCompletion(context.Background()) {
		t.Fatal("completion should be claimed when no stop is pending")
	}
	if scanner.isScanning {
		t.Fatal("scanner should reject new stop requests after completion is claimed")
	}
}

func TestClaimCompletionRejectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scanner := &Scanner{isScanning: true}
	if scanner.claimCompletion(ctx) {
		t.Fatal("completion should not win after context cancellation")
	}
}
