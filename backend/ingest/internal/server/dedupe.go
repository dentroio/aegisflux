package server

import "sync"

const defaultDuplicateTrackerLimit = 10000

// duplicateTracker tracks recently accepted event IDs to avoid replaying the
// same visibility event into downstream processors during lab ingest.
type duplicateTracker struct {
	mu    sync.Mutex
	limit int
	seen  map[string]struct{}
	order []string
}

func newDuplicateTracker(limit int) *duplicateTracker {
	if limit <= 0 {
		limit = defaultDuplicateTrackerLimit
	}
	return &duplicateTracker{
		limit: limit,
		seen:  make(map[string]struct{}, limit),
		order: make([]string, 0, limit),
	}
}

func (d *duplicateTracker) has(eventID string) bool {
	if eventID == "" {
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	_, exists := d.seen[eventID]
	return exists
}

func (d *duplicateTracker) add(eventID string) {
	if eventID == "" {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.seen[eventID]; exists {
		return
	}

	d.seen[eventID] = struct{}{}
	d.order = append(d.order, eventID)

	if len(d.order) > d.limit {
		oldest := d.order[0]
		delete(d.seen, oldest)
		d.order = d.order[1:]
	}
}
