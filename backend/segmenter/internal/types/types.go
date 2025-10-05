package types

import "time"

// SegmentationGoal represents different segmentation objectives
type SegmentationGoal string

const (
	GoalReduceLateralMovement SegmentationGoal = "reduce_lateral_movement"
	GoalCompliance            SegmentationGoal = "compliance"
	GoalPerformance           SegmentationGoal = "performance"
	GoalCost                  SegmentationGoal = "cost"
	GoalSecurity              SegmentationGoal = "security"
)

// SegmentationStrategy represents different segmentation approaches
type SegmentationStrategy string

const (
	StrategyMicrosegmentation SegmentationStrategy = "microsegmentation"
	StrategyZeroTrust         SegmentationStrategy = "zero_trust"
	StrategyTraditional       SegmentationStrategy = "traditional"
	StrategyHybrid            SegmentationStrategy = "hybrid"
)

// SecurityLevel represents the security level of a network segment
type SecurityLevel string

const (
	SecurityLevelHigh   SecurityLevel = "high"
	SecurityLevelMedium SecurityLevel = "medium"
	SecurityLevelLow    SecurityLevel = "low"
)

// ImplementationMode represents how segmentation should be implemented
type ImplementationMode string

const (
	ModeConservative ImplementationMode = "conservative"
	ModeBalanced     ImplementationMode = "balanced"
	ModeAggressive   ImplementationMode = "aggressive"
	ModeTest         ImplementationMode = "test"
)

// Host represents a network host
type Host struct {
	ID       string   `json:"id"`
	IP       string   `json:"ip"`
	Labels   []string `json:"labels"`
	Services []string `json:"services"`
}

// TrafficFlow represents network traffic flow data
type TrafficFlow struct {
	SourceHost        string    `json:"source_host"`
	DestinationHost   string    `json:"destination_host"`
	DestinationPort   int       `json:"destination_port"`
	Protocol          string    `json:"protocol"`
	BytesTransferred  int64     `json:"bytes_transferred"`
	PacketCount       int64     `json:"packet_count"`
	Timestamp         time.Time `json:"timestamp"`
}

// NetworkSegment represents a network segment
type NetworkSegment struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Hosts         []*Host       `json:"hosts"`
	SecurityLevel SecurityLevel `json:"security_level"`
	CIDR          string        `json:"cidr,omitempty"`
	VLAN          int           `json:"vlan,omitempty"`
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Port        string `json:"port"`
	Protocol    string `json:"protocol"`
	Action      string `json:"action"` // allow, deny, drop
	Priority    int    `json:"priority"`
}

// SegmentationProposal represents a segmentation proposal
type SegmentationProposal struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	Strategy         SegmentationStrategy  `json:"strategy"`
	Segments         []*NetworkSegment     `json:"segments"`
	FirewallRules    []*FirewallRule       `json:"firewall_rules"`
	RiskReduction    float64               `json:"risk_reduction"`
	ImplementationComplexity int           `json:"implementation_complexity"`
	CostEstimate     float64               `json:"cost_estimate,omitempty"`
}

// SegmentationProposalRequest represents a request to generate segmentation proposals
type SegmentationProposalRequest struct {
	Hosts        []*Host        `json:"hosts"`
	TrafficData  []*TrafficFlow `json:"traffic_data"`
	TrafficPeriod string        `json:"traffic_period"`
	Goals        []SegmentationGoal `json:"goals"`
	Constraints  map[string]interface{} `json:"constraints,omitempty"`
}

// SegmentationProposalResponse represents the response with segmentation proposals
type SegmentationProposalResponse struct {
	ProposalID      string             `json:"proposal_id"`
	GeneratedAt     time.Time          `json:"generated_at"`
	Topology        *NetworkTopology   `json:"topology"`
	Proposals       []*SegmentationProposal `json:"proposals"`
	RiskAssessment  *RiskAssessment    `json:"risk_assessment"`
}

// NetworkTopology represents the analyzed network topology
type NetworkTopology struct {
	Hosts       []*Host       `json:"hosts"`
	Connections []*Connection `json:"connections"`
}

// Connection represents a network connection
type Connection struct {
	SourceHost      string    `json:"source_host"`
	DestinationHost string    `json:"destination_host"`
	Port           int       `json:"port"`
	Protocol       string    `json:"protocol"`
	TrafficVolume  int64     `json:"traffic_volume"`
	PacketCount    int64     `json:"packet_count"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
	RiskLevel      string    `json:"risk_level"`
}

// RiskAssessment represents a risk assessment
type RiskAssessment struct {
	OverallRisk string   `json:"overall_risk"`
	RiskFactors []string `json:"risk_factors"`
	RiskScore   float64  `json:"risk_score"`
}

// SegmentationPlanRequest represents a request to create a segmentation plan
type SegmentationPlanRequest struct {
	ProposalID         string             `json:"proposal_id"`
	Proposal           *SegmentationProposal `json:"proposal"`
	ImplementationMode ImplementationMode `json:"implementation_mode"`
	Timeline           *time.Duration     `json:"timeline,omitempty"`
	Constraints        map[string]interface{} `json:"constraints,omitempty"`
}

// ImplementationStep represents a step in the implementation plan
type ImplementationStep struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Order       int           `json:"order"`
	Duration    time.Duration `json:"duration"`
	Dependencies []string     `json:"dependencies"`
	RiskLevel   string        `json:"risk_level"`
	Rollbackable bool         `json:"rollbackable"`
}

// RollbackPlan represents a rollback plan
type RollbackPlan struct {
	Steps []*ImplementationStep `json:"steps"`
	EstimatedDuration time.Duration `json:"estimated_duration"`
}

// SegmentationPlanResponse represents the response with a segmentation plan
type SegmentationPlanResponse struct {
	PlanID              string                 `json:"plan_id"`
	ProposalID          string                 `json:"proposal_id"`
	CreatedAt           time.Time              `json:"created_at"`
	ImplementationMode  ImplementationMode     `json:"implementation_mode"`
	Steps               []*ImplementationStep  `json:"steps"`
	Policies            []interface{}          `json:"policies"`
	RollbackPlan        *RollbackPlan          `json:"rollback_plan"`
	EstimatedDuration   time.Duration          `json:"estimated_duration"`
	RiskLevel           string                 `json:"risk_level"`
}
