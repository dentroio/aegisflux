package policy

// SegmentPolicy represents a policy for a network segment
type SegmentPolicy struct {
	ID          string
	Name        string
	SegmentID   string
	Rules       []*PolicyRule
	Priority    int
	Enabled     bool
}

// PolicyRule represents a policy rule
type PolicyRule struct {
	ID          string
	Name        string
	Source      string
	Destination string
	Port        string
	Protocol    string
	Action      string // allow, deny, drop, log
	Priority    int
	Enabled     bool
}

