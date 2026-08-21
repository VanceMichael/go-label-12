package compliance

import (
	"fmt"
	"sync"
	"time"

	"go-base/internal/domain"
)

type ReviewLease struct {
	TenantID   string
	GroupID    string
	ReviewerID string
	AcquiredAt time.Time
}

type LeaseTable struct {
	mu     sync.RWMutex
	leases map[string]ReviewLease
}

func NewLeaseTable() *LeaseTable {
	return &LeaseTable{leases: make(map[string]ReviewLease)}
}

func (t *LeaseTable) Acquire(lease ReviewLease) error {
	if lease.TenantID == "" || lease.GroupID == "" || lease.ReviewerID == "" || lease.AcquiredAt.IsZero() {
		return fmt.Errorf("%w: review lease identity", domain.ErrInvalid)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	key := leaseKey(lease.TenantID, lease.GroupID)
	if current, exists := t.leases[key]; exists {
		return fmt.Errorf("%w: group review is held by %s", domain.ErrConflict, current.ReviewerID)
	}
	t.leases[key] = lease
	return nil
}

func (t *LeaseTable) Release(tenantID, groupID, reviewerID string) error {
	if tenantID == "" || groupID == "" || reviewerID == "" {
		return fmt.Errorf("%w: review lease release", domain.ErrInvalid)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	key := leaseKey(tenantID, groupID)
	current, exists := t.leases[key]
	if !exists {
		return domain.ErrNotFound
	}
	if current.ReviewerID != reviewerID {
		return domain.ErrForbidden
	}
	delete(t.leases, key)
	return nil
}

func (t *LeaseTable) Current(tenantID, groupID string) (ReviewLease, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	lease, exists := t.leases[leaseKey(tenantID, groupID)]
	if !exists {
		return ReviewLease{}, domain.ErrNotFound
	}
	return lease, nil
}

func (t *LeaseTable) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.leases)
}

func leaseKey(tenantID, groupID string) string {
	return tenantID + "\x00" + groupID
}
