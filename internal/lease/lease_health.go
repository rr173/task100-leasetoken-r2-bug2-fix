package lease

import "task100-leasetoken/internal/model"

// LeaseHealth explains the state a caller should expect before mutating a
// lease, including remaining lifetime and the current fencing generation.
type LeaseHealth struct {
	LeaseID       string `json:"lease_id"`
	Resource      string `json:"resource"`
	Status        string `json:"status"`
	RemainingSecs int64  `json:"remaining_secs"`
	FencingToken  int64  `json:"fencing_token"`
	CanRenew      bool   `json:"can_renew"`
}

func (m *Manager) LeaseHealth(id string) (LeaseHealth, error) {
	l, err := m.GetLease(id)
	if err != nil {
		return LeaseHealth{}, err
	}
	now := m.now()
	remaining := l.ExpiresAt - now
	// CanRenew mirrors the Renew gating so the health report never advertises
	// a renewal that Renew would reject. The lease must be active (not a
	// released/expired terminal row), must not be logically expired, and the
	// renew window must be open: remaining TTL at or below the configured
	// fraction of ttl_seconds. Before the window opens the answer is false;
	// once the lease has passed its expires_at the window has closed and the
	// answer is false again.
	threshold := int64(float64(l.TTLSeconds) * RenewWindowFraction)
	canRenew := !model.IsTerminal(l.Status) && remaining >= 0 && remaining <= threshold
	return LeaseHealth{LeaseID: l.LeaseID, Resource: l.Resource, Status: l.Status, RemainingSecs: remaining, FencingToken: l.FencingToken, CanRenew: canRenew}, nil
}
