package compliance

import (
	"context"
	"fmt"
	"time"

	"go-base/internal/domain"
)

type ReleaseRequest struct {
	TenantID    string
	GroupID     string
	BatchID     string
	ReviewerID  string
	RequestedAt time.Time
}

type ReleaseDecision struct {
	GroupID   string
	Allowed   bool
	Reasons   []string
	CheckedAt time.Time
}

type ReleaseChecker interface {
	Check(context.Context, ReleaseRequest) (ReleaseDecision, error)
}

type ReleaseCoordinator struct {
	Leases  *LeaseTable
	Checker ReleaseChecker
	Now     func() time.Time
}

func (c ReleaseCoordinator) ReviewRelease(ctx context.Context, request ReleaseRequest) (ReleaseDecision, error) {
	if err := validateReleaseRequest(request); err != nil {
		return ReleaseDecision{}, err
	}
	if c.Leases == nil || c.Checker == nil {
		return ReleaseDecision{}, fmt.Errorf("%w: release dependencies", domain.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return ReleaseDecision{}, err
	}

	lease := ReviewLease{
		TenantID:   request.TenantID,
		GroupID:    request.GroupID,
		ReviewerID: request.ReviewerID,
		AcquiredAt: c.now(),
	}
	if err := c.Leases.Acquire(lease); err != nil {
		return ReleaseDecision{}, err
	}
	defer func() {
		_ = c.Leases.Release(request.TenantID, request.GroupID, request.ReviewerID)
	}()

	decision, err := c.Checker.Check(context.Background(), request)
	if err != nil {
		return ReleaseDecision{}, fmt.Errorf("check release eligibility: %w", err)
	}
	if decision.GroupID != request.GroupID {
		return ReleaseDecision{}, fmt.Errorf("%w: release decision group", domain.ErrConflict)
	}
	if decision.CheckedAt.IsZero() {
		decision.CheckedAt = c.now()
	}
	decision.Reasons = append([]string(nil), decision.Reasons...)
	return decision, nil
}

func validateReleaseRequest(request ReleaseRequest) error {
	if request.TenantID == "" || request.GroupID == "" || request.BatchID == "" || request.ReviewerID == "" {
		return fmt.Errorf("%w: release request identity", domain.ErrInvalid)
	}
	if request.RequestedAt.IsZero() {
		return fmt.Errorf("%w: release request time", domain.ErrInvalid)
	}
	return nil
}

func (c ReleaseCoordinator) now() time.Time {
	if c.Now != nil {
		if now := c.Now(); !now.IsZero() {
			return now.UTC()
		}
	}
	return time.Now().UTC()
}
