package segmenter

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strings"
	"time"

	"aegisflux/backend/segmenter/internal/network"
	"aegisflux/backend/segmenter/internal/policy"
	"aegisflux/backend/segmenter/internal/types"
)

// Segmenter handles network segmentation logic
type Segmenter struct {
	logger *slog.Logger
}

// NewSegmenter creates a new segmenter instance
func NewSegmenter(logger *slog.Logger) *Segmenter {
	return &Segmenter{
		logger: logger,
	}
}

// ProposeSegmentation analyzes network traffic and proposes segmentation boundaries
func (s *Segmenter) ProposeSegmentation(ctx context.Context, req *types.SegmentationProposalRequest) (*types.SegmentationProposalResponse, error) {
	s.logger.Info("Processing segmentation proposal request",
		"host_count", len(req.Hosts),
		"traffic_period", req.TrafficPeriod,
		"segmentation_goals", req.Goals)

	// Parse hosts and extract network information
	hosts := make([]*network.Host, 0, len(req.Hosts))
	for _, h := range req.Hosts {
		host := &network.Host{
			ID:           h.ID,
			IP:           h.IP,
			Labels:       h.Labels,
			Services:     h.Services,
			Criticality:  s.assessCriticality(h),
			TrustLevel:   s.assessTrustLevel(h),
		}
		hosts = append(hosts, host)
	}

	// Analyze network topology
	topology := s.analyzeTopology(hosts, req.TrafficData)

	// Generate segmentation proposals based on goals
	proposals := s.generateProposals(topology, req.Goals)

	// Calculate risk reduction and complexity scores
	for _, proposal := range proposals {
		proposal.RiskReduction = s.calculateRiskReduction(topology, proposal)
		proposal.ImplementationComplexity = s.calculateComplexity(proposal)
	}

	// Sort proposals by risk reduction vs complexity ratio
	sort.Slice(proposals, func(i, j int) bool {
		scoreI := proposals[i].RiskReduction / float64(proposals[i].ImplementationComplexity)
		scoreJ := proposals[j].RiskReduction / float64(proposals[j].ImplementationComplexity)
		return scoreI > scoreJ
	})

	response := &types.SegmentationProposalResponse{
		ProposalID:    fmt.Sprintf("proposal-%d", time.Now().Unix()),
		GeneratedAt:   time.Now(),
		Topology:      s.convertTopology(topology),
		Proposals:     proposals,
		RiskAssessment: s.generateRiskAssessment(topology),
	}

	s.logger.Info("Generated segmentation proposals",
		"proposal_count", len(proposals),
		"topology_size", len(topology.Connections))

	return response, nil
}

// CreateSegmentationPlan converts a proposal into an implementation plan
func (s *Segmenter) CreateSegmentationPlan(ctx context.Context, req *types.SegmentationPlanRequest) (*types.SegmentationPlanResponse, error) {
	s.logger.Info("Creating segmentation plan",
		"proposal_id", req.ProposalID,
		"implementation_mode", req.ImplementationMode)

	// Validate proposal
	if req.Proposal == nil {
		return nil, fmt.Errorf("proposal is required")
	}

	// Generate implementation steps
	steps := s.generateImplementationSteps(req.Proposal, req.ImplementationMode)

	// Create policy rules for each segment
	policies := s.generateSegmentPolicies(req.Proposal)

	// Generate rollback plan
	rollbackPlan := s.generateRollbackPlan(req.Proposal)

	response := &types.SegmentationPlanResponse{
		PlanID:            fmt.Sprintf("plan-%d", time.Now().Unix()),
		ProposalID:        req.ProposalID,
		CreatedAt:         time.Now(),
		ImplementationMode: req.ImplementationMode,
		Steps:             steps,
		Policies:          s.convertPoliciesToInterface(policies),
		RollbackPlan:      rollbackPlan,
		EstimatedDuration: s.estimateDuration(steps),
		RiskLevel:         s.assessPlanRisk(steps),
	}

	s.logger.Info("Created segmentation plan",
		"plan_id", response.PlanID,
		"step_count", len(steps),
		"policy_count", len(policies))

	return response, nil
}

// assessCriticality determines the criticality level of a host
func (s *Segmenter) assessCriticality(host *types.Host) network.CriticalityLevel {
	// Check for critical labels
	criticalLabels := []string{"role:db", "role:control-plane", "role:monitoring", "critical", "production"}
	for _, label := range host.Labels {
		for _, critical := range criticalLabels {
			if strings.Contains(strings.ToLower(label), critical) {
				return network.CriticalityHigh
			}
		}
	}

	// Check for database services
	for _, service := range host.Services {
		if strings.Contains(strings.ToLower(service), "database") ||
			strings.Contains(strings.ToLower(service), "db") {
			return network.CriticalityHigh
		}
	}

	// Check for web services (medium criticality)
	for _, service := range host.Services {
		if strings.Contains(strings.ToLower(service), "http") ||
			strings.Contains(strings.ToLower(service), "web") {
			return network.CriticalityMedium
		}
	}

	return network.CriticalityLow
}

// assessTrustLevel determines the trust level of a host
func (s *Segmenter) assessTrustLevel(host *types.Host) network.TrustLevel {
	// Check for security labels
	securityLabels := []string{"secure", "trusted", "internal"}
	for _, label := range host.Labels {
		for _, secure := range securityLabels {
			if strings.Contains(strings.ToLower(label), secure) {
				return network.TrustHigh
			}
		}
	}

	// Check for external indicators
	externalLabels := []string{"external", "dmz", "public"}
	for _, label := range host.Labels {
		for _, external := range externalLabels {
			if strings.Contains(strings.ToLower(label), external) {
				return network.TrustLow
			}
		}
	}

	// Default to medium trust for internal hosts
	return network.TrustMedium
}

// analyzeTopology analyzes network topology and communication patterns
func (s *Segmenter) analyzeTopology(hosts []*network.Host, trafficData []*types.TrafficFlow) *network.Topology {
	topology := &network.Topology{
		Hosts:       make(map[string]*network.Host),
		Connections: make(map[string]*network.Connection),
	}

	// Add hosts to topology
	for _, host := range hosts {
		topology.Hosts[host.ID] = host
	}

	// Analyze traffic flows
	for _, flow := range trafficData {
		connectionKey := fmt.Sprintf("%s->%s:%d", flow.SourceHost, flow.DestinationHost, flow.DestinationPort)
		
		if conn, exists := topology.Connections[connectionKey]; exists {
			conn.TrafficVolume += flow.BytesTransferred
			conn.PacketCount += flow.PacketCount
			conn.LastSeen = flow.Timestamp.Unix()
		} else {
			topology.Connections[connectionKey] = &network.Connection{
				SourceHost:      flow.SourceHost,
				DestinationHost: flow.DestinationHost,
				Port:           flow.DestinationPort,
				Protocol:       flow.Protocol,
				TrafficVolume:  flow.BytesTransferred,
				PacketCount:    flow.PacketCount,
			FirstSeen:      flow.Timestamp.Unix(),
			LastSeen:       flow.Timestamp.Unix(),
				RiskLevel:      s.assessConnectionRisk(flow),
			}
		}
	}

	// Calculate host connectivity metrics
	for _, host := range topology.Hosts {
		host.InboundConnections = s.countConnections(topology.Connections, host.ID, true)
		host.OutboundConnections = s.countConnections(topology.Connections, host.ID, false)
		host.TotalTraffic = s.calculateHostTraffic(topology.Connections, host.ID)
	}

	return topology
}

// generateProposals generates segmentation proposals based on topology and goals
func (s *Segmenter) generateProposals(topology *network.Topology, goals []types.SegmentationGoal) []*types.SegmentationProposal {
	var proposals []*types.SegmentationProposal

	// Generate proposals based on different segmentation strategies
	if s.hasGoal(goals, types.GoalReduceLateralMovement) {
		proposals = append(proposals, s.generateMicrosegmentationProposal(topology))
		proposals = append(proposals, s.generateZeroTrustProposal(topology))
	}

	if s.hasGoal(goals, types.GoalCompliance) {
		proposals = append(proposals, s.generateComplianceProposal(topology))
	}

	if s.hasGoal(goals, types.GoalPerformance) {
		proposals = append(proposals, s.generatePerformanceProposal(topology))
	}

	return proposals
}

// generateMicrosegmentationProposal creates a microsegmentation proposal
func (s *Segmenter) generateMicrosegmentationProposal(topology *network.Topology) *types.SegmentationProposal {
	segments := make([]*types.NetworkSegment, 0)

	// Group hosts by criticality and trust level
	criticalityGroups := make(map[network.CriticalityLevel][]*network.Host)
	trustGroups := make(map[network.TrustLevel][]*network.Host)

	for _, host := range topology.Hosts {
		criticalityGroups[host.Criticality] = append(criticalityGroups[host.Criticality], host)
		trustGroups[host.TrustLevel] = append(trustGroups[host.TrustLevel], host)
	}

	// Create segments based on criticality
	for criticality, hosts := range criticalityGroups {
		if len(hosts) > 0 {
			segment := &types.NetworkSegment{
				ID:          fmt.Sprintf("criticality-%s", strings.ToLower(string(criticality))),
				Name:        fmt.Sprintf("%s Criticality Segment", strings.Title(string(criticality))),
				Description: fmt.Sprintf("Hosts with %s criticality", strings.ToLower(string(criticality))),
				Hosts:       s.convertHosts(hosts),
				SecurityLevel: s.mapCriticalityToSecurityLevel(criticality),
			}
			segments = append(segments, segment)
		}
	}

	// Create DMZ segment for external-facing services
	dmzHosts := make([]*network.Host, 0)
	for _, host := range topology.Hosts {
		if host.TrustLevel == network.TrustLow {
			dmzHosts = append(dmzHosts, host)
		}
	}

	if len(dmzHosts) > 0 {
		dmzSegment := &types.NetworkSegment{
			ID:            "dmz",
			Name:          "DMZ Segment",
			Description:   "External-facing services and untrusted hosts",
			Hosts:         s.convertHosts(dmzHosts),
			SecurityLevel: types.SecurityLevelLow,
		}
		segments = append(segments, dmzSegment)
	}

	return &types.SegmentationProposal{
		ID:           fmt.Sprintf("microseg-%d", time.Now().Unix()),
		Name:         "Microsegmentation Proposal",
		Description:  "Fine-grained segmentation based on host criticality and trust levels",
		Strategy:     types.StrategyMicrosegmentation,
		Segments:     segments,
		FirewallRules: s.generateMicrosegmentationRules(segments),
	}
}

// generateZeroTrustProposal creates a zero-trust segmentation proposal
func (s *Segmenter) generateZeroTrustProposal(topology *network.Topology) *types.SegmentationProposal {
	segments := make([]*types.NetworkSegment, 0)

	// Create individual segments for each host (maximum isolation)
	for _, host := range topology.Hosts {
		segment := &types.NetworkSegment{
			ID:            fmt.Sprintf("host-%s", host.ID),
			Name:          fmt.Sprintf("Host %s Segment", host.ID),
			Description:   fmt.Sprintf("Isolated segment for host %s", host.ID),
			Hosts:         []*types.Host{{ID: host.ID, IP: host.IP, Labels: host.Labels}},
			SecurityLevel: types.SecurityLevelHigh,
		}
		segments = append(segments, segment)
	}

	return &types.SegmentationProposal{
		ID:           fmt.Sprintf("zerotrust-%d", time.Now().Unix()),
		Name:         "Zero Trust Proposal",
		Description:  "Maximum isolation with individual segments for each host",
		Strategy:     types.StrategyZeroTrust,
		Segments:     segments,
		FirewallRules: s.generateZeroTrustRules(segments),
	}
}

// Helper functions for proposal generation
func (s *Segmenter) hasGoal(goals []types.SegmentationGoal, target types.SegmentationGoal) bool {
	for _, goal := range goals {
		if goal == target {
			return true
		}
	}
	return false
}

func (s *Segmenter) assessConnectionRisk(flow *types.TrafficFlow) network.RiskLevel {
	// High risk for external connections
	if s.isExternalIP(flow.DestinationHost) {
		return network.RiskHigh
	}

	// Medium risk for high-volume connections
	if flow.BytesTransferred > 100*1024*1024 { // 100MB
		return network.RiskMedium
	}

	return network.RiskLow
}

func (s *Segmenter) isExternalIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return true // Treat invalid IPs as external
	}

	// Check for private network ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(parsedIP) {
			return false
		}
	}

	return true
}

func (s *Segmenter) countConnections(connections map[string]*network.Connection, hostID string, inbound bool) int {
	count := 0
	for _, conn := range connections {
		if inbound && conn.DestinationHost == hostID {
			count++
		} else if !inbound && conn.SourceHost == hostID {
			count++
		}
	}
	return count
}

func (s *Segmenter) calculateHostTraffic(connections map[string]*network.Connection, hostID string) int64 {
	var totalTraffic int64
	for _, conn := range connections {
		if conn.SourceHost == hostID || conn.DestinationHost == hostID {
			totalTraffic += conn.TrafficVolume
		}
	}
	return totalTraffic
}

func (s *Segmenter) convertHosts(hosts []*network.Host) []*types.Host {
	apiHosts := make([]*types.Host, len(hosts))
	for i, host := range hosts {
		apiHosts[i] = &types.Host{
			ID:       host.ID,
			IP:       host.IP,
			Labels:   host.Labels,
			Services: host.Services,
		}
	}
	return apiHosts
}

func (s *Segmenter) mapCriticalityToSecurityLevel(criticality network.CriticalityLevel) types.SecurityLevel {
	switch criticality {
	case network.CriticalityHigh:
		return types.SecurityLevelHigh
	case network.CriticalityMedium:
		return types.SecurityLevelMedium
	default:
		return types.SecurityLevelLow
	}
}

func (s *Segmenter) convertTopology(topology *network.Topology) *types.NetworkTopology {
	return &types.NetworkTopology{
		Hosts:       s.convertHostsMap(topology.Hosts),
		Connections: s.convertConnectionsMap(topology.Connections),
	}
}

func (s *Segmenter) convertHostsMap(hosts map[string]*network.Host) []*types.Host {
	apiHosts := make([]*types.Host, 0, len(hosts))
	for _, host := range hosts {
		apiHosts = append(apiHosts, &types.Host{
			ID:       host.ID,
			IP:       host.IP,
			Labels:   host.Labels,
			Services: host.Services,
		})
	}
	return apiHosts
}

func (s *Segmenter) convertConnectionsMap(connections map[string]*network.Connection) []*types.Connection {
	apiConnections := make([]*types.Connection, 0, len(connections))
	for _, conn := range connections {
		apiConnections = append(apiConnections, &types.Connection{
			SourceHost:      conn.SourceHost,
			DestinationHost: conn.DestinationHost,
			Port:           conn.Port,
			Protocol:       conn.Protocol,
			TrafficVolume:  conn.TrafficVolume,
			PacketCount:    conn.PacketCount,
		FirstSeen:      time.Unix(conn.FirstSeen, 0),
		LastSeen:       time.Unix(conn.LastSeen, 0),
			RiskLevel:      string(conn.RiskLevel),
		})
	}
	return apiConnections
}

// Additional helper methods for proposal generation, risk calculation, etc.
func (s *Segmenter) generateMicrosegmentationRules(segments []*types.NetworkSegment) []*types.FirewallRule {
	// Implementation for microsegmentation rules
	return []*types.FirewallRule{}
}

func (s *Segmenter) generateZeroTrustRules(segments []*types.NetworkSegment) []*types.FirewallRule {
	// Implementation for zero trust rules
	return []*types.FirewallRule{}
}

func (s *Segmenter) calculateRiskReduction(topology *network.Topology, proposal *types.SegmentationProposal) float64 {
	// Calculate potential risk reduction from this proposal
	return 0.75 // Placeholder
}

func (s *Segmenter) calculateComplexity(proposal *types.SegmentationProposal) int {
	// Calculate implementation complexity
	return len(proposal.Segments) * 2 // Placeholder
}

func (s *Segmenter) generateRiskAssessment(topology *network.Topology) *types.RiskAssessment {
	// Generate risk assessment
	return &types.RiskAssessment{
		OverallRisk: "Medium",
		RiskFactors: []string{"High traffic volume", "External connections"},
	}
}

func (s *Segmenter) generateImplementationSteps(proposal *types.SegmentationProposal, mode types.ImplementationMode) []*types.ImplementationStep {
	// Generate implementation steps
	return []*types.ImplementationStep{}
}

func (s *Segmenter) generateSegmentPolicies(proposal *types.SegmentationProposal) []*policy.SegmentPolicy {
	// Generate segment policies
	return []*policy.SegmentPolicy{}
}

func (s *Segmenter) generateRollbackPlan(proposal *types.SegmentationProposal) *types.RollbackPlan {
	// Generate rollback plan
	return &types.RollbackPlan{}
}

func (s *Segmenter) estimateDuration(steps []*types.ImplementationStep) time.Duration {
	// Estimate implementation duration
	return time.Hour * 2 // Placeholder
}

func (s *Segmenter) assessPlanRisk(steps []*types.ImplementationStep) string {
	// Assess plan risk level
	return "Medium"
}

func (s *Segmenter) generateComplianceProposal(topology *network.Topology) *types.SegmentationProposal {
	// Generate compliance-focused proposal
	return &types.SegmentationProposal{}
}

func (s *Segmenter) generatePerformanceProposal(topology *network.Topology) *types.SegmentationProposal {
	// Generate performance-focused proposal
	return &types.SegmentationProposal{}
}

func (s *Segmenter) convertPoliciesToInterface(policies []*policy.SegmentPolicy) []interface{} {
	interfaces := make([]interface{}, len(policies))
	for i, p := range policies {
		interfaces[i] = p
	}
	return interfaces
}
