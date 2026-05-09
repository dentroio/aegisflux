package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

var aiFindingHint = regexp.MustCompile(`(?i)ai|agent|browser|model`)

// --- Dashboard summary ---

type dashboardSummaryResponse struct {
	OK            bool              `json:"ok"`
	GeneratedAtMS int64             `json:"generated_at_ms"`
	Devices       []json.RawMessage `json:"devices"`
	Model         dashboardModelDTO `json:"model"`
}

type dashboardModelDTO struct {
	TotalDevices          int                       `json:"totalDevices"`
	OnlineDevices         int                       `json:"onlineDevices"`
	OfflineDevices        int                       `json:"offlineDevices"`
	MaxRisk               int                       `json:"maxRisk"`
	AISignals             int                       `json:"aiSignals"`
	EventCount            int                       `json:"eventCount"`
	ExtensionCount        int                       `json:"extensionCount"`
	SaseCount             int                       `json:"saseCount"`
	Extensions            []any                     `json:"extensions"`
	Sase                  []any                     `json:"sase"`
	HealthyCollectorPairs int                       `json:"healthyCollectorPairs"`
	MaxCPUPercent         *float64                  `json:"maxCpuPercent"`
	AvgCPUPercent         *float64                  `json:"avgCpuPercent"`
	MaxMemoryRSSMb        *float64                  `json:"maxMemoryRssMb"`
	CollectorStatuses     []collectorStatusSnapshot `json:"collectorStatuses"`
	Performance           []performanceSnapshot     `json:"performance"`
}

type collectorStatusSnapshot struct {
	DeviceID     string `json:"device_id"`
	Collector    string `json:"collector"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	ReceivedAtMS int64  `json:"received_at_ms"`
}

type performanceSnapshot struct {
	DeviceID           string   `json:"device_id"`
	OS                 string   `json:"os,omitempty"`
	ProcessCPUPercent  *float64 `json:"process_cpu_percent,omitempty"`
	ProcessMemoryRSSMb *float64 `json:"process_memory_rss_mb,omitempty"`
	CollectorRuntimeMS int64    `json:"collector_runtime_ms,omitempty"`
	CollectorName      string   `json:"collector_name,omitempty"`
	EventQueueDepth    int      `json:"event_queue_depth,omitempty"`
	SpoolBytes         int64    `json:"spool_bytes,omitempty"`
	PackEvalRuntimeMS  *float64 `json:"pack_eval_runtime_ms,omitempty"`
	ReceivedAtMS       int64    `json:"received_at_ms,omitempty"`
}

type perfPayload struct {
	OS                 string   `json:"os"`
	ProcessCPUPercent  *float64 `json:"process_cpu_percent"`
	ProcessMemoryRSSMb *float64 `json:"process_memory_rss_mb"`
	CollectorRuntimeMS int64    `json:"collector_runtime_ms"`
	CollectorName      string   `json:"collector_name"`
	EventQueueDepth    int      `json:"event_queue_depth"`
	SpoolBytes         int64    `json:"spool_bytes"`
	PackEvalRuntimeMS  *float64 `json:"pack_eval_runtime_ms"`
}

type collectorPayload struct {
	Collector string `json:"collector"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

func (s *IngestServer) handleSummaryDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}

	deviceLimit, err := parseQueryLimit(r.URL.Query().Get("device_limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("device_limit") == "" {
		deviceLimit = 80
	}

	ctx := r.Context()
	devices, err := s.store.ListDevices(ctx, visibilityDeviceFilter{Limit: deviceLimit})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	eventsGeneral, _ := s.store.Query(ctx, visibilityQueryFilter{Limit: 120})
	eventsExt, _ := s.store.Query(ctx, visibilityQueryFilter{EventType: "aegis.browser_extension.observed", Limit: 80})
	eventsSase, _ := s.store.Query(ctx, visibilityQueryFilter{EventType: "aegis.sase_component.observed", Limit: 80})
	eventsColl, _ := s.store.Query(ctx, visibilityQueryFilter{EventType: "aegis.collector.status", Limit: 120})
	eventsPerf, _ := s.store.Query(ctx, visibilityQueryFilter{EventType: "aegis.agent.performance", Limit: 120})

	merged := uniqueEventsRecentFirst(eventsGeneral, eventsExt, eventsSase, eventsColl, eventsPerf)

	findings := collectFindingRecords(ctx, s, "", 80)

	now := time.Now().UnixMilli()
	model := buildDashboardModelDTO(devices, merged, findings, now)

	rawDevices, err := json.Marshal(devices)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var wrapped []json.RawMessage
	if err := json.Unmarshal(rawDevices, &wrapped); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, dashboardSummaryResponse{
		OK:            true,
		GeneratedAtMS: now,
		Devices:       wrapped,
		Model:         model,
	})
}

func uniqueEventsRecentFirst(batches ...[]visibilityEvent) []visibilityEvent {
	byID := make(map[string]visibilityEvent)
	for _, batch := range batches {
		for _, event := range batch {
			if event.EventID == "" {
				continue
			}
			byID[event.EventID] = event
		}
	}
	out := make([]visibilityEvent, 0, len(byID))
	for _, e := range byID {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		ri := out[i].ReceivedAtMS
		if ri == 0 {
			ri = out[i].TimestampMS
		}
		rj := out[j].ReceivedAtMS
		if rj == 0 {
			rj = out[j].TimestampMS
		}
		return ri > rj
	})
	return out
}

func collectFindingRecords(ctx context.Context, s *IngestServer, deviceID string, cap int) []findingRecord {
	events, err := s.store.Query(ctx, visibilityQueryFilter{
		DeviceID: deviceID,
		Limit:    maxVisibilityQueryLimit,
	})
	if err != nil {
		return nil
	}
	findings := make([]findingRecord, 0, cap)
	for _, event := range events {
		if !isFindingEventType(event.EventType) {
			continue
		}
		record, err := event.toFindingRecord()
		if err != nil {
			continue
		}
		findings = append(findings, record)
		if len(findings) >= cap {
			break
		}
	}
	return findings
}

func buildDashboardModelDTO(devices []visibilityDeviceRecord, mergedEvents []visibilityEvent, findings []findingRecord, nowMS int64) dashboardModelDTO {
	freshWindow := int64(5 * 60 * 1000)
	online := 0
	for _, d := range devices {
		if nowMS-d.LastSeenMS < freshWindow {
			online++
		}
	}
	maxRisk := 0
	for _, f := range findings {
		if f.RiskScore > maxRisk {
			maxRisk = f.RiskScore
		}
	}
	aiSignals := 0
	for _, f := range findings {
		if aiShapedFinding(f) {
			aiSignals++
		}
	}

	var extensions []any
	var saseRows []any
	var collStatuses []collectorStatusSnapshot
	var perfRows []performanceSnapshot
	healthyPairs := make(map[string]struct{})
	var cpuSamples []float64
	var memSamples []float64

	for _, event := range mergedEvents {
		switch event.EventType {
		case "aegis.browser_extension.observed":
			var payload map[string]any
			_ = json.Unmarshal(event.Payload, &payload)
			row := map[string]any{"device_id": event.DeviceID}
			for k, v := range payload {
				row[k] = v
			}
			extensions = append(extensions, row)
		case "aegis.sase_component.observed":
			var payload map[string]any
			_ = json.Unmarshal(event.Payload, &payload)
			row := map[string]any{"device_id": event.DeviceID}
			for k, v := range payload {
				row[k] = v
			}
			saseRows = append(saseRows, row)
		case "aegis.collector.status":
			var p collectorPayload
			_ = json.Unmarshal(event.Payload, &p)
			recv := event.ReceivedAtMS
			if recv == 0 {
				recv = event.TimestampMS
			}
			collStatuses = append(collStatuses, collectorStatusSnapshot{
				DeviceID:     event.DeviceID,
				Collector:    p.Collector,
				Status:       p.Status,
				Message:      p.Message,
				ReceivedAtMS: recv,
			})
			if strings.EqualFold(strings.TrimSpace(p.Status), "healthy") {
				healthyPairs[fmt.Sprintf("%s:%s", event.DeviceID, p.Collector)] = struct{}{}
			}
		case "aegis.agent.performance":
			var p perfPayload
			_ = json.Unmarshal(event.Payload, &p)
			recv := event.ReceivedAtMS
			if recv == 0 {
				recv = event.TimestampMS
			}
			perfRows = append(perfRows, performanceSnapshot{
				DeviceID:           event.DeviceID,
				OS:                 p.OS,
				ProcessCPUPercent:  p.ProcessCPUPercent,
				ProcessMemoryRSSMb: p.ProcessMemoryRSSMb,
				CollectorRuntimeMS: p.CollectorRuntimeMS,
				CollectorName:      p.CollectorName,
				EventQueueDepth:    p.EventQueueDepth,
				SpoolBytes:         p.SpoolBytes,
				PackEvalRuntimeMS:  p.PackEvalRuntimeMS,
				ReceivedAtMS:       recv,
			})
			if p.ProcessCPUPercent != nil {
				cpuSamples = append(cpuSamples, *p.ProcessCPUPercent)
			}
			if p.ProcessMemoryRSSMb != nil {
				memSamples = append(memSamples, *p.ProcessMemoryRSSMb)
			}
		}
	}

	var maxCPU, avgCPU, maxMem *float64
	if len(cpuSamples) > 0 {
		m := cpuSamples[0]
		sum := 0.0
		for _, v := range cpuSamples {
			if v > m {
				m = v
			}
			sum += v
		}
		maxCPU = &m
		a := sum / float64(len(cpuSamples))
		avgCPU = &a
	}
	if len(memSamples) > 0 {
		m := memSamples[0]
		for _, v := range memSamples {
			if v > m {
				m = v
			}
		}
		maxMem = &m
	}

	return dashboardModelDTO{
		TotalDevices:          len(devices),
		OnlineDevices:         online,
		OfflineDevices:        maxInt(0, len(devices)-online),
		MaxRisk:               maxRisk,
		AISignals:             aiSignals,
		EventCount:            len(mergedEvents),
		ExtensionCount:        len(extensions),
		SaseCount:             len(saseRows),
		Extensions:            extensions,
		Sase:                  saseRows,
		HealthyCollectorPairs: len(healthyPairs),
		MaxCPUPercent:         maxCPU,
		AvgCPUPercent:         avgCPU,
		MaxMemoryRSSMb:        maxMem,
		CollectorStatuses:     collStatuses,
		Performance:           perfRows,
	}
}

func aiShapedFinding(f findingRecord) bool {
	var title, class string
	if f.Title != nil {
		title = *f.Title
	}
	if f.Classification != nil {
		class = *f.Classification
	}
	hay := strings.ToLower(title + " " + class + " " + strings.Join(f.DetectedPatterns, " "))
	return aiFindingHint.MatchString(hay)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --- Device detail bundle ---

type deviceSummaryResponse struct {
	OK                bool                     `json:"ok"`
	Devices           []visibilityDeviceRecord `json:"devices"`
	Events            []visibilityEvent        `json:"events"`
	Processes         []processRecord          `json:"processes"`
	Flows             []flowRecord             `json:"flows"`
	DNS               []dnsRecord              `json:"dns"`
	Findings          []findingRecord          `json:"findings"`
	ExtensionEvents   []visibilityEvent        `json:"extension_events"`
	SaseEvents        []visibilityEvent        `json:"sase_events"`
	CollectorEvents   []visibilityEvent        `json:"collector_events"`
	PerformanceEvents []visibilityEvent        `json:"performance_events"`
}

func (s *IngestServer) handleSummaryDevice(w http.ResponseWriter, r *http.Request) {
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
	devices, err := s.store.ListDevices(ctx, visibilityDeviceFilter{Limit: 120})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	events, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, Limit: 120})

	procEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, Limit: maxVisibilityQueryLimit})
	processes := filterProcesses(procEvents, 80)

	flowEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, Limit: maxVisibilityQueryLimit})
	flows := filterFlows(flowEvents, 80)

	dnsEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, Limit: maxVisibilityQueryLimit})
	dnsRows := filterDNS(dnsEvents, 80)

	findings := collectFindingRecords(ctx, s, deviceID, 80)

	extEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, EventType: "aegis.browser_extension.observed", Limit: 80})
	saseEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, EventType: "aegis.sase_component.observed", Limit: 80})
	collEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, EventType: "aegis.collector.status", Limit: 80})
	perfEvents, _ := s.store.Query(ctx, visibilityQueryFilter{DeviceID: deviceID, EventType: "aegis.agent.performance", Limit: 80})

	writeJSON(w, http.StatusOK, deviceSummaryResponse{
		OK:                true,
		Devices:           devices,
		Events:            events,
		Processes:         processes,
		Flows:             flows,
		DNS:               dnsRows,
		Findings:          findings,
		ExtensionEvents:   extEvents,
		SaseEvents:        saseEvents,
		CollectorEvents:   collEvents,
		PerformanceEvents: perfEvents,
	})
}

func filterProcesses(events []visibilityEvent, limit int) []processRecord {
	out := make([]processRecord, 0, limit)
	for _, event := range events {
		if event.EventType != "aegis.process.started" && event.EventType != "aegis.process.ended" {
			continue
		}
		record, err := event.toProcessRecord()
		if err != nil {
			continue
		}
		out = append(out, record)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func filterFlows(events []visibilityEvent, limit int) []flowRecord {
	out := make([]flowRecord, 0, limit)
	for _, event := range events {
		if event.EventType != "aegis.flow.started" && event.EventType != "aegis.flow.ended" {
			continue
		}
		record, err := event.toFlowRecord()
		if err != nil {
			continue
		}
		out = append(out, record)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func filterDNS(events []visibilityEvent, limit int) []dnsRecord {
	out := make([]dnsRecord, 0, limit)
	for _, event := range events {
		if event.EventType != "aegis.dns.observed" {
			continue
		}
		record, err := event.toDNSRecord()
		if err != nil {
			continue
		}
		out = append(out, record)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// --- Inventory bundle ---

type inventorySummaryResponse struct {
	OK         bool                        `json:"ok"`
	Devices    []inventoryDeviceSummaryRow `json:"devices"`
	EventsExt  []visibilityEvent           `json:"events_ext"`
	EventsSase []visibilityEvent           `json:"events_sase"`
	DNS        []dnsRecord                 `json:"dns"`
	Processes  []processRecord             `json:"processes"`
	Findings   []findingRecord             `json:"findings"`
}

type inventoryDeviceSummaryRow struct {
	DeviceID      string `json:"device_id"`
	Source        string `json:"source,omitempty"`
	LastSeenMS    int64  `json:"last_seen_ms,omitempty"`
	AgentID       string `json:"agent_id,omitempty"`
	SensorVersion string `json:"sensor_version,omitempty"`
}

func (s *IngestServer) handleSummaryInventory(w http.ResponseWriter, r *http.Request) {
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
	devices, err := s.store.ListDevices(ctx, visibilityDeviceFilter{Limit: 180})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	eventsExt, _ := s.store.Query(ctx, visibilityQueryFilter{EventType: "aegis.browser_extension.observed", Limit: 220})
	eventsSase, _ := s.store.Query(ctx, visibilityQueryFilter{EventType: "aegis.sase_component.observed", Limit: 220})

	dnsEvents, _ := s.store.Query(ctx, visibilityQueryFilter{Limit: maxVisibilityQueryLimit})
	dnsRows := filterDNS(dnsEvents, 220)

	procEvents, _ := s.store.Query(ctx, visibilityQueryFilter{Limit: maxVisibilityQueryLimit})
	processes := filterProcesses(procEvents, 220)

	findings := collectFindingRecords(ctx, s, "", 120)

	devOut := make([]inventoryDeviceSummaryRow, 0, len(devices))
	for _, d := range devices {
		devOut = append(devOut, inventoryDeviceSummaryRow{
			DeviceID:      d.DeviceID,
			Source:        d.Source,
			LastSeenMS:    d.LastSeenMS,
			AgentID:       d.AgentID,
			SensorVersion: d.SensorVersion,
		})
	}

	writeJSON(w, http.StatusOK, inventorySummaryResponse{
		OK:         true,
		Devices:    devOut,
		EventsExt:  eventsExt,
		EventsSase: eventsSase,
		DNS:        dnsRows,
		Processes:  processes,
		Findings:   findings,
	})
}
