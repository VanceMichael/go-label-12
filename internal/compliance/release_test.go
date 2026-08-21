package compliance

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-base/internal/domain"
)

type blockingReleaseChecker struct {
	started chan struct{}
	release chan struct{}
}

func (c *blockingReleaseChecker) Check(ctx context.Context, request ReleaseRequest) (ReleaseDecision, error) {
	close(c.started)
	select {
	case <-ctx.Done():
		return ReleaseDecision{}, ctx.Err()
	case <-c.release:
		return ReleaseDecision{GroupID: request.GroupID, Allowed: true, CheckedAt: request.RequestedAt}, nil
	}
}

func TestCancelledReleaseReviewReturnsAndFreesGroupLease(t *testing.T) {
	now := time.Date(2026, 8, 21, 3, 0, 0, 0, time.UTC)
	checker := &blockingReleaseChecker{started: make(chan struct{}), release: make(chan struct{})}
	leases := NewLeaseTable()
	coordinator := ReleaseCoordinator{Leases: leases, Checker: checker, Now: func() time.Time { return now }}
	request := ReleaseRequest{
		TenantID: "farm-east", GroupID: "group-12", BatchID: "review-12", ReviewerID: "biosecurity-1", RequestedAt: now,
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := coordinator.ReviewRelease(ctx, request)
		result <- err
	}()
	<-checker.started
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled review error = %v, want context.Canceled", err)
		}
		if _, err := leases.Current(request.TenantID, request.GroupID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("lease after cancellation = %v, want released", err)
		}
	case <-time.After(100 * time.Millisecond):
		_, secondErr := coordinator.ReviewRelease(context.Background(), ReleaseRequest{
			TenantID: request.TenantID, GroupID: request.GroupID, BatchID: "review-13", ReviewerID: "biosecurity-2", RequestedAt: now,
		})
		close(checker.release)
		<-result
		if !errors.Is(secondErr, domain.ErrConflict) {
			t.Fatalf("cancelled review stayed blocked; follow-up error = %v, want lease conflict", secondErr)
		}
		t.Fatal("cancelled release review stayed blocked and retained the group lease")
	}
}
