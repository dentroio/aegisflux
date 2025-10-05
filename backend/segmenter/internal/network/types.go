package network

// CriticalityLevel represents the criticality level of a host
type CriticalityLevel string

const (
	CriticalityHigh   CriticalityLevel = "high"
	CriticalityMedium CriticalityLevel = "medium"
	CriticalityLow    CriticalityLevel = "low"
)

// TrustLevel represents the trust level of a host
type TrustLevel string

const (
	TrustHigh   TrustLevel = "high"
	TrustMedium TrustLevel = "medium"
	TrustLow    TrustLevel = "low"
)

// RiskLevel represents the risk level of a connection
type RiskLevel string

const (
	RiskHigh   RiskLevel = "high"
	RiskMedium RiskLevel = "medium"
	RiskLow    RiskLevel = "low"
)

// Host represents a network host with extended information
type Host struct {
	ID                  string
	IP                  string
	Labels              []string
	Services            []string
	Criticality         CriticalityLevel
	TrustLevel          TrustLevel
	InboundConnections  int
	OutboundConnections int
	TotalTraffic        int64
}

// Connection represents a network connection with risk assessment
type Connection struct {
	SourceHost      string
	DestinationHost string
	Port           int
	Protocol       string
	TrafficVolume  int64
	PacketCount    int64
	FirstSeen      int64 // Unix timestamp
	LastSeen       int64 // Unix timestamp
	RiskLevel      RiskLevel
}

// Topology represents the analyzed network topology
type Topology struct {
	Hosts       map[string]*Host
	Connections map[string]*Connection
}

