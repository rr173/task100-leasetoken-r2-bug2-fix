package lease

type AcquireAdvice struct {
	Resource     string `json:"resource"`
	Allowed      bool   `json:"allowed"`
	Locked       bool   `json:"locked"`
	CurrentToken int64  `json:"current_token"`
	Reason       string `json:"reason"`
}

// AcquireAdvice mirrors the manager's acquire rules without mutating state, so
// a client can learn whether an acquisition attempt would succeed before paying
// for it. The decision matches Acquire exactly: a missing resource is
// creatable, a locked resource is refused with ErrResourceLocked, and a
// resource held by an active, unexpired lease is refused with ErrConflict. A
// lease whose status is still active but whose expires_at has already passed is
// treated as available, since Acquire would evict it in place.
func (m *Manager) AcquireAdvice(resource string) (AcquireAdvice, error) {
	tx, err := m.store.BeginTx()
	if err != nil {
		return AcquireAdvice{}, err
	}
	defer rollback(tx)

	row, found, err := m.store.GetResource(tx, resource)
	if err != nil {
		return AcquireAdvice{}, err
	}
	if !found {
		return AcquireAdvice{Resource: resource, Allowed: true, Reason: "resource will be created"}, tx.Commit()
	}
	if row.Locked {
		return AcquireAdvice{Resource: resource, Locked: true, CurrentToken: row.FencingToken, Reason: ErrResourceLocked.Error()}, tx.Commit()
	}
	// A real Acquire would conflict here; mirror that decision rather than
	// advertising a free resource while it is still actively held.
	now := m.now()
	if existing, found, err := m.store.GetActiveLeaseByResource(tx, resource); err != nil {
		return AcquireAdvice{}, err
	} else if found && existing.ExpiresAt > now {
		return AcquireAdvice{Resource: resource, CurrentToken: row.FencingToken, Reason: ErrConflict.Error()}, tx.Commit()
	}
	return AcquireAdvice{Resource: resource, Allowed: true, CurrentToken: row.FencingToken, Reason: "resource is available for an acquisition attempt"}, tx.Commit()
}
