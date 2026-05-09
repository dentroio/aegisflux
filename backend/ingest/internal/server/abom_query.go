package server

// Agent Bill of Materials (ABOM) aggregation.
//
// ABOM turns raw visibility telemetry (process events, browser extensions,
// SASE/SSE component observations, DNS lookups, findings) into a fleet-wide
// inventory of AI-capable tools and supporting evidence references. The
// product story is "what AI capability exists on this endpoint, and what
// proves it" — see WO-PROD-001 for design intent.
//
// All confidence values are heuristic. We never claim completeness across the
// entire AI ecosystem. False positives are expected when names overlap with
// generic process or domain names.

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	abomConfidenceHigh   = "high"
	abomConfidenceMedium = "medium"
	abomConfidenceLow    = "low"
)

// ABOM categories. Keep this taxonomy stable — UI and docs depend on it.
const (
	abomCategoryAIDesktopApp        = "ai_desktop_app"
	abomCategoryBrowserAIExtension  = "browser_ai_extension"
	abomCategoryCodingAgent         = "coding_agent"
	abomCategoryCLIAgent            = "cli_agent"
	abomCategoryMCPEndpoint         = "mcp_endpoint"
	abomCategoryLocalModelRuntime   = "local_model_runtime"
	abomCategoryModelGateway        = "model_gateway"
	abomCategoryUnknownAIAutomation = "unknown_ai_automation"
)

// abomItem is the canonical ABOM record returned by the API.
type abomItem struct {
	ID                string   `json:"id"`
	Category          string   `json:"category"`
	Product           string   `json:"product"`
	CapabilityTags    []string `json:"capability_tags"`
	Confidence        string   `json:"confidence"`
	DeviceIDs         []string `json:"device_ids"`
	UserContext       string   `json:"user_context,omitempty"`
	EvidenceRefs      []string `json:"evidence_refs"`
	FirstSeenMS       int64    `json:"first_seen_ms"`
	LastSeenMS        int64    `json:"last_seen_ms"`
	RecommendedReview string   `json:"recommended_review"`
}

type abomFleetResponse struct {
	OK            bool        `json:"ok"`
	GeneratedAtMS int64       `json:"generated_at_ms"`
	TotalItems    int         `json:"total_items"`
	CategoryCount map[string]int `json:"category_count"`
	Items         []abomItem  `json:"items"`
	EmptyHelp     string      `json:"empty_help,omitempty"`
}

type abomDeviceResponse struct {
	OK            bool       `json:"ok"`
	DeviceID      string     `json:"device_id"`
	GeneratedAtMS int64      `json:"generated_at_ms"`
	TotalItems    int        `json:"total_items"`
	Items         []abomItem `json:"items"`
	EmptyHelp     string     `json:"empty_help,omitempty"`
}

// detection patterns. These are intentionally conservative — better to miss
// some signals than to mis-classify generic processes.
var (
	abomDesktopAppPattern = regexp.MustCompile(`(?i)\b(chatgpt(\s|-)?desktop|claude(\s|-)?desktop|copilot(\s|-)?desktop|gemini(\s|-)?desktop|perplexity(\s|-)?desktop)\b`)
	abomCodingAgentPattern = regexp.MustCompile(`(?i)\b(cursor|cursoragent|aider|continue\.dev|continue|tabnine|sweep|github(\s|-)?copilot|cody)\b`)
	abomCLIAgentPattern    = regexp.MustCompile(`(?i)\b(codex|claude(\s|-)?code|claudecode|gpt(\s|-)?cli|gemini(\s|-)?cli|llm\s|^llm$|aichat|aider)\b`)
	abomMCPPattern         = regexp.MustCompile(`(?i)(modelcontextprotocol|^mcp[-_]server|/mcp[-_]server|mcp\.)`)
	abomLocalRuntimePattern = regexp.MustCompile(`(?i)\b(ollama|llama\.cpp|llamacpp|vllm|llamafile|localai|gpt4all|lm\s?studio|lmstudio|llama-server)\b`)
	abomGatewayDomainPattern = regexp.MustCompile(`(?i)(api\.openai\.com|api\.anthropic\.com|api\.cohere\.ai|api\.mistral\.ai|generativelanguage\.googleapis\.com|api\.perplexity\.ai|bedrock\.[a-z0-9-]+\.amazonaws\.com|api\.together\.xyz|api\.groq\.com|portkey\.ai|litellm)`)
	abomBrowserAIPattern    = regexp.MustCompile(`(?i)(chatgpt|openai|claude|anthropic|gemini|copilot|perplexity|monica|ai\b)`)
	abomCLIBinaryHints      = regexp.MustCompile(`(?i)/(usr|opt|home|root)/.*/(codex|cursor|aider|claude(-code)?|gemini|llm|aichat|continue|cody)$`)
)

// handleABOMFleet returns ABOM aggregated across the fleet.
func (s *IngestServer) handleABOMFleet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()

	devices, err := s.store.ListDevices(ctx, visibilityDeviceFilter{Limit: 220})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	processes, _ := s.collectProcesses(ctx, "", maxVisibilityQueryLimit)
	dnsRows, _ := s.collectDNS(ctx, "", maxVisibilityQueryLimit)
	findings := collectFindingRecords(ctx, s, "", 200)
	extEvents, _ := s.store.Query(ctx, visibilityQueryFilter{EventType: "aegis.browser_extension.observed", Limit: 220})
	saseEvents, _ := s.store.Query(ctx, visibilityQueryFilter{EventType: "aegis.sase_component.observed", Limit: 220})

	items := buildABOMItems(processes, dnsRows, findings, extEvents, saseEvents)

	count := make(map[string]int, len(items))
	for _, item := range items {
		count[item.Category]++
	}

	resp := abomFleetResponse{
		OK:            true,
		GeneratedAtMS: time.Now().UnixMilli(),
		TotalItems:    len(items),
		CategoryCount: count,
		Items:         items,
	}
	if len(items) == 0 {
		resp.EmptyHelp = abomEmptyHelp(len(devices))
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleABOMDevice returns ABOM scoped to a single device.
func (s *IngestServer) handleABOMDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if deviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	processes, _ := s.collectProcesses(ctx, deviceID, maxVisibilityQueryLimit)
	dnsRows, _ := s.collectDNS(ctx, deviceID, maxVisibilityQueryLimit)
	findings := collectFindingRecords(ctx, s, deviceID, 200)
	extEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, EventType: "aegis.browser_extension.observed", Limit: 220})
	saseEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, EventType: "aegis.sase_component.observed", Limit: 220})

	items := buildABOMItems(processes, dnsRows, findings, extEvents, saseEvents)
	deviceItems := make([]abomItem, 0, len(items))
	for _, item := range items {
		if containsString(item.DeviceIDs, deviceID) {
			deviceItems = append(deviceItems, item)
		}
	}

	resp := abomDeviceResponse{
		OK:            true,
		DeviceID:      deviceID,
		GeneratedAtMS: time.Now().UnixMilli(),
		TotalItems:    len(deviceItems),
		Items:         deviceItems,
	}
	if len(deviceItems) == 0 {
		resp.EmptyHelp = "No AI-capable signals observed for this endpoint yet. ABOM populates from process, browser extension, SASE/SSE, DNS, and finding evidence."
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *IngestServer) collectProcesses(ctx context.Context, deviceID string, cap int) ([]processRecord, error) {
	events, err := s.store.Query(ctx, visibilityQueryFilter{
		DeviceID: deviceID,
		Limit:    maxVisibilityQueryLimit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]processRecord, 0, cap)
	for _, event := range events {
		if event.EventType != "aegis.process.started" && event.EventType != "aegis.process.ended" {
			continue
		}
		record, err := event.toProcessRecord()
		if err != nil {
			continue
		}
		out = append(out, record)
		if len(out) >= cap {
			break
		}
	}
	return out, nil
}

func (s *IngestServer) collectDNS(ctx context.Context, deviceID string, cap int) ([]dnsRecord, error) {
	events, err := s.store.Query(ctx, visibilityQueryFilter{
		DeviceID:  deviceID,
		EventType: "aegis.dns.observed",
		Limit:     maxVisibilityQueryLimit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]dnsRecord, 0, cap)
	for _, event := range events {
		if event.EventType != "aegis.dns.observed" {
			continue
		}
		record, err := event.toDNSRecord()
		if err != nil {
			continue
		}
		out = append(out, record)
		if len(out) >= cap {
			break
		}
	}
	return out, nil
}

// buildABOMItems is the pure aggregation function used by handlers and tests.
func buildABOMItems(
	processes []processRecord,
	dnsRows []dnsRecord,
	findings []findingRecord,
	extensionEvents []visibilityEvent,
	saseEvents []visibilityEvent,
) []abomItem {
	bag := newABOMBag()

	for _, proc := range processes {
		bag.addProcess(proc)
	}
	for _, row := range dnsRows {
		bag.addDNS(row)
	}
	for _, ev := range extensionEvents {
		bag.addExtension(ev)
	}
	for _, ev := range saseEvents {
		bag.addSase(ev)
	}
	for _, finding := range findings {
		bag.addFinding(finding)
	}

	out := bag.toItems()
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		if out[i].Product != out[j].Product {
			return out[i].Product < out[j].Product
		}
		return out[i].LastSeenMS > out[j].LastSeenMS
	})
	return out
}

type abomBag struct {
	items map[string]*abomItem
}

func newABOMBag() *abomBag {
	return &abomBag{items: make(map[string]*abomItem)}
}

func (b *abomBag) upsert(category, product string, capabilityTags []string, confidence string, deviceID string, evidence string, recommendedReview string, ts int64) {
	key := category + "::" + strings.ToLower(strings.TrimSpace(product))
	existing, ok := b.items[key]
	if !ok {
		existing = &abomItem{
			ID:                abomItemID(category, product),
			Category:          category,
			Product:           product,
			CapabilityTags:    uniqueStrings(capabilityTags),
			Confidence:        confidence,
			DeviceIDs:         []string{},
			EvidenceRefs:      []string{},
			FirstSeenMS:       ts,
			LastSeenMS:        ts,
			RecommendedReview: recommendedReview,
		}
		b.items[key] = existing
	}
	if deviceID != "" {
		existing.DeviceIDs = appendUnique(existing.DeviceIDs, deviceID)
	}
	if evidence != "" {
		existing.EvidenceRefs = appendUnique(existing.EvidenceRefs, evidence)
		if len(existing.EvidenceRefs) > 16 {
			existing.EvidenceRefs = existing.EvidenceRefs[:16]
		}
	}
	if len(capabilityTags) > 0 {
		existing.CapabilityTags = uniqueStrings(append(existing.CapabilityTags, capabilityTags...))
	}
	if ts > 0 {
		if existing.FirstSeenMS == 0 || ts < existing.FirstSeenMS {
			existing.FirstSeenMS = ts
		}
		if ts > existing.LastSeenMS {
			existing.LastSeenMS = ts
		}
	}
	// Confidence promotion: high > medium > low. Repeated observations across
	// multiple devices also push toward higher confidence.
	existing.Confidence = mergeConfidence(existing.Confidence, confidence)
	if len(existing.DeviceIDs) >= 3 && existing.Confidence != abomConfidenceHigh {
		existing.Confidence = abomConfidenceHigh
	}
}

func (b *abomBag) addProcess(proc processRecord) {
	name := stringOrEmpty(proc.Name)
	path := stringOrEmpty(proc.Path)
	cmd := stringOrEmpty(proc.CommandLine)
	hay := strings.ToLower(name + " " + path + " " + cmd)
	ts := proc.TimestampMS
	deviceID := proc.DeviceID
	evidence := evidenceRefForProcess(proc)

	switch {
	case abomMCPPattern.MatchString(hay):
		b.upsert(abomCategoryMCPEndpoint,
			productNameForProcess(name, path, "MCP server"),
			[]string{"protocol:mcp", "tool-call"},
			abomConfidenceMedium,
			deviceID, evidence,
			"Confirm whether this MCP server is approved by platform/security and what tools it exposes.", ts)
	case abomLocalRuntimePattern.MatchString(hay):
		b.upsert(abomCategoryLocalModelRuntime,
			productNameForProcess(name, path, "Local model runtime"),
			[]string{"local-llm"},
			abomConfidenceHigh,
			deviceID, evidence,
			"Verify approved use, model provenance, and whether the runtime exposes a local API.", ts)
	case abomDesktopAppPattern.MatchString(hay):
		b.upsert(abomCategoryAIDesktopApp,
			productNameForProcess(name, path, "AI desktop app"),
			[]string{"chat", "vendor-cloud"},
			abomConfidenceHigh,
			deviceID, evidence,
			"Confirm enterprise approval and tenant boundary configuration.", ts)
	case abomCodingAgentPattern.MatchString(hay):
		b.upsert(abomCategoryCodingAgent,
			productNameForProcess(name, path, "Coding agent"),
			[]string{"code-edit", "vendor-cloud"},
			abomConfidenceMedium,
			deviceID, evidence,
			"Verify repo scope, output review process, and any data sent to a model provider.", ts)
	case abomCLIAgentPattern.MatchString(hay) || (path != "" && abomCLIBinaryHints.MatchString(path)):
		b.upsert(abomCategoryCLIAgent,
			productNameForProcess(name, path, "AI CLI agent"),
			[]string{"shell", "tool-call"},
			abomConfidenceMedium,
			deviceID, evidence,
			"Review what the CLI agent can run, whether shell automation is approved, and where output goes.", ts)
	}
}

func (b *abomBag) addDNS(record dnsRecord) {
	q := record.Query
	if q == "" {
		return
	}
	hay := strings.ToLower(q)
	ts := record.TimestampMS
	deviceID := record.DeviceID
	evidence := evidenceRefForDNS(record)

	if abomGatewayDomainPattern.MatchString(hay) {
		b.upsert(abomCategoryModelGateway,
			gatewayProductForHost(q),
			[]string{"egress", "vendor-cloud"},
			abomConfidenceHigh,
			deviceID, evidence,
			"Confirm allowed model providers and whether traffic should route through an enterprise gateway.", ts)
		return
	}
	if abomBrowserAIPattern.MatchString(hay) {
		// DNS alone is weak signal — record as unknown automation candidate.
		b.upsert(abomCategoryUnknownAIAutomation,
			fmt.Sprintf("AI-related DNS: %s", q),
			[]string{"egress"},
			abomConfidenceLow,
			deviceID, evidence,
			"Investigate which process initiated this lookup and whether the destination is enterprise-approved.", ts)
	}
}

func (b *abomBag) addExtension(event visibilityEvent) {
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return
	}
	name := stringFromAny(payload["name"])
	if strings.TrimSpace(name) == "" {
		name = stringFromAny(payload["extension_id"])
	}
	hay := strings.ToLower(name + " " + stringFromAny(payload["description"]))
	if !abomBrowserAIPattern.MatchString(hay) {
		return
	}
	hostPerms := joinStringSlice(payload["host_permissions"])
	tags := []string{"browser"}
	if strings.Contains(strings.ToLower(hostPerms), "*://*/*") {
		tags = append(tags, "broad-host-access")
	}
	confidence := abomConfidenceMedium
	if strings.Contains(strings.ToLower(hostPerms), "*://*/*") {
		confidence = abomConfidenceHigh
	}
	b.upsert(abomCategoryBrowserAIExtension,
		strings.TrimSpace(name),
		tags,
		confidence,
		event.DeviceID,
		fmt.Sprintf("event:%s", event.EventID),
		"Review extension permissions and which sites it can access on managed browsers.",
		eventTimestamp(event))
}

func (b *abomBag) addSase(event visibilityEvent) {
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return
	}
	component := stringFromAny(payload["component_type"])
	if !strings.EqualFold(component, "model_gateway") && !strings.EqualFold(component, "ai_gateway") {
		return
	}
	product := strings.TrimSpace(stringFromAny(payload["vendor"]) + " " + stringFromAny(payload["product"]))
	if product == "" {
		product = "Enterprise model gateway"
	}
	b.upsert(abomCategoryModelGateway,
		product,
		[]string{"vendor-cloud", "control-point"},
		abomConfidenceHigh,
		event.DeviceID,
		fmt.Sprintf("event:%s", event.EventID),
		"Confirm the gateway is the documented enforcement point and audit which providers are allowed.",
		eventTimestamp(event))
}

func (b *abomBag) addFinding(record findingRecord) {
	title := stringOrEmpty(record.Title)
	classification := stringOrEmpty(record.Classification)
	patterns := strings.Join(record.DetectedPatterns, " ")
	hay := strings.ToLower(title + " " + classification + " " + patterns)
	if !abomBrowserAIPattern.MatchString(hay) {
		return
	}
	if record.DeviceID == "" {
		return
	}
	b.upsert(abomCategoryUnknownAIAutomation,
		fmt.Sprintf("AI-related finding: %s", strings.TrimSpace(title)),
		[]string{"finding"},
		abomConfidenceLow,
		record.DeviceID,
		fmt.Sprintf("finding:%s", stringOrEmpty(record.FindingID)),
		"Open the finding to confirm whether AI activity caused the alert.",
		record.TimestampMS)
}

func (b *abomBag) toItems() []abomItem {
	out := make([]abomItem, 0, len(b.items))
	for _, item := range b.items {
		clone := *item
		clone.DeviceIDs = append([]string{}, clone.DeviceIDs...)
		clone.EvidenceRefs = append([]string{}, clone.EvidenceRefs...)
		clone.CapabilityTags = append([]string{}, clone.CapabilityTags...)
		out = append(out, clone)
	}
	return out
}

func abomItemID(category, product string) string {
	sum := sha1.Sum([]byte(category + "|" + strings.ToLower(strings.TrimSpace(product))))
	return fmt.Sprintf("abom-%x", sum[:6])
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := values[:0]
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return append([]string{}, out...)
}

func mergeConfidence(current, incoming string) string {
	rank := func(c string) int {
		switch c {
		case abomConfidenceHigh:
			return 3
		case abomConfidenceMedium:
			return 2
		case abomConfidenceLow:
			return 1
		default:
			return 0
		}
	}
	if rank(incoming) > rank(current) {
		return incoming
	}
	return current
}

func evidenceRefForProcess(p processRecord) string {
	name := stringOrEmpty(p.Name)
	if name == "" {
		name = "process"
	}
	return fmt.Sprintf("process:%s pid=%d device=%s", name, p.PID, p.DeviceID)
}

func evidenceRefForDNS(r dnsRecord) string {
	answer := strings.Join(r.Answers, ",")
	return fmt.Sprintf("dns:%s answers=%s device=%s", r.Query, answer, r.DeviceID)
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func joinStringSlice(value any) string {
	switch slice := value.(type) {
	case []any:
		parts := make([]string, 0, len(slice))
		for _, item := range slice {
			parts = append(parts, stringFromAny(item))
		}
		return strings.Join(parts, " ")
	case []string:
		return strings.Join(slice, " ")
	}
	return ""
}

func eventTimestamp(event visibilityEvent) int64 {
	if event.ReceivedAtMS > 0 {
		return event.ReceivedAtMS
	}
	return event.TimestampMS
}

func productNameForProcess(name, path, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if strings.TrimSpace(path) != "" {
		base := path
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		return strings.TrimSpace(base)
	}
	return fallback
}

func gatewayProductForHost(host string) string {
	clean := strings.ToLower(strings.TrimSpace(host))
	switch {
	case strings.Contains(clean, "openai"):
		return "OpenAI API"
	case strings.Contains(clean, "anthropic"):
		return "Anthropic API"
	case strings.Contains(clean, "googleapis"):
		return "Google AI API"
	case strings.Contains(clean, "cohere"):
		return "Cohere API"
	case strings.Contains(clean, "mistral"):
		return "Mistral API"
	case strings.Contains(clean, "perplexity"):
		return "Perplexity API"
	case strings.Contains(clean, "bedrock"):
		return "AWS Bedrock"
	case strings.Contains(clean, "together"):
		return "Together AI"
	case strings.Contains(clean, "groq"):
		return "Groq"
	case strings.Contains(clean, "portkey") || strings.Contains(clean, "litellm"):
		return "LLM Gateway"
	}
	return clean
}

func abomEmptyHelp(deviceCount int) string {
	if deviceCount == 0 {
		return "No reporting endpoints yet. Connect at least one lab agent so ABOM can populate from process, DNS, browser extension, and finding evidence."
	}
	return "No AI-capable signals observed yet. ABOM populates from process, DNS, browser extension, SASE/SSE, and finding evidence as agents report."
}
