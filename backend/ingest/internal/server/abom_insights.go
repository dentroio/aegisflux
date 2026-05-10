package server

// ABOM fleet insights (WO-GROWTH-001).
//
// Insights turn the raw ABOM inventory into operator-centered callouts:
//
//   - newly_observed                : items with first_seen >= since cutoff
//   - newly_observed_high_attention : newly observed items where any
//                                     associated device has open findings
//   - high_confidence               : confidence == high (trust this row)
//   - low_confidence_needs_review   : confidence == low (needs more evidence)
//   - widespread                    : item observed on >= widespread threshold
//                                     of devices in the fleet
//   - stale                         : last_seen older than stale cutoff
//
// Endpoint hotspots highlight devices with the densest AI footprint or
// concentrated low-confidence rows that deserve manual review.
//
// The aggregator is pure and tested in isolation; the HTTP handler stitches
// it onto the ingest store.

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	abomDefaultNewSinceMS  int64 = 24 * 60 * 60 * 1000        // 24h
	abomDefaultStaleMS     int64 = 7 * 24 * 60 * 60 * 1000    // 7d
	abomWidespreadMinCount int   = 3
	abomWidespreadFraction       = 0.3 // 30% of fleet devices
)

// ABOM insight section ids — exported via JSON.
const (
	abomInsightNew                  = "newly_observed"
	abomInsightNewHighAttention     = "newly_observed_high_attention"
	abomInsightHighConfidence       = "high_confidence"
	abomInsightLowConfidenceReview  = "low_confidence_needs_review"
	abomInsightWidespread           = "widespread"
	abomInsightStale                = "stale"
)

type abomInsightItem struct {
	ID             string   `json:"id"`
	Category       string   `json:"category"`
	Product        string   `json:"product"`
	Confidence     string   `json:"confidence"`
	DeviceIDs      []string `json:"device_ids"`
	DeviceCount    int      `json:"device_count"`
	FirstSeenMS    int64    `json:"first_seen_ms"`
	LastSeenMS     int64    `json:"last_seen_ms"`
	Reason         string   `json:"reason"`
	HighAttention  bool     `json:"high_attention,omitempty"`
}

type abomInsightSection struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Detail      string            `json:"detail"`
	Threshold   string            `json:"threshold,omitempty"`
	Items       []abomInsightItem `json:"items"`
	Total       int               `json:"total"`
}

type abomEndpointHotspot struct {
	DeviceID       string   `json:"device_id"`
	ItemCount      int      `json:"item_count"`
	HighConfidence int      `json:"high_confidence_count"`
	LowConfidence  int      `json:"low_confidence_count"`
	Categories     []string `json:"categories"`
	LastSeenMS     int64    `json:"last_seen_ms"`
	HighAttention  bool     `json:"high_attention"`
	Reason         string   `json:"reason"`
}

type abomInsightsResponse struct {
	OK                  bool                  `json:"ok"`
	GeneratedAtMS       int64                 `json:"generated_at_ms"`
	WindowMS            int64                 `json:"window_ms"`
	StaleAfterMS        int64                 `json:"stale_after_ms"`
	FleetSize           int                   `json:"fleet_size"`
	HighAttentionDevices []string             `json:"high_attention_devices"`
	Sections            []abomInsightSection  `json:"sections"`
	Hotspots            []abomEndpointHotspot `json:"hotspots"`
	EmptyHelp           string                `json:"empty_help,omitempty"`
}

func (s *IngestServer) handleABOMInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}

	now := time.Now().UnixMilli()
	q := r.URL.Query()
	windowMS := parsePositiveDuration(q.Get("since_ms"), abomDefaultNewSinceMS)
	staleMS := parsePositiveDuration(q.Get("stale_after_ms"), abomDefaultStaleMS)

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

	highAttention := highAttentionDeviceIDs(findings)

	insights := buildABOMInsights(items, len(devices), highAttention, abomInsightOptions{
		Now:          now,
		WindowMS:     windowMS,
		StaleMS:      staleMS,
	})

	resp := abomInsightsResponse{
		OK:                   true,
		GeneratedAtMS:        now,
		WindowMS:             windowMS,
		StaleAfterMS:         staleMS,
		FleetSize:            len(devices),
		HighAttentionDevices: insights.HighAttentionDevices,
		Sections:             insights.Sections,
		Hotspots:             insights.Hotspots,
	}
	if len(items) == 0 {
		resp.EmptyHelp = abomEmptyHelp(len(devices))
	}
	writeJSON(w, http.StatusOK, resp)
}

type abomInsightOptions struct {
	Now      int64
	WindowMS int64
	StaleMS  int64
}

type abomInsightsBundle struct {
	HighAttentionDevices []string
	Sections             []abomInsightSection
	Hotspots             []abomEndpointHotspot
}

// buildABOMInsights is the pure aggregator used by the handler and tests.
func buildABOMInsights(items []abomItem, fleetSize int, highAttentionDevices []string, opts abomInsightOptions) abomInsightsBundle {
	if opts.Now == 0 {
		opts.Now = time.Now().UnixMilli()
	}
	if opts.WindowMS <= 0 {
		opts.WindowMS = abomDefaultNewSinceMS
	}
	if opts.StaleMS <= 0 {
		opts.StaleMS = abomDefaultStaleMS
	}

	highSet := make(map[string]struct{}, len(highAttentionDevices))
	for _, id := range highAttentionDevices {
		if id != "" {
			highSet[id] = struct{}{}
		}
	}

	cutoffNew := opts.Now - opts.WindowMS
	cutoffStale := opts.Now - opts.StaleMS
	widespreadThreshold := abomWidespreadMinCount
	if fleetSize > 0 {
		minByFraction := int(abomWidespreadFraction*float64(fleetSize) + 0.5)
		if minByFraction > widespreadThreshold {
			widespreadThreshold = minByFraction
		}
	}

	sections := make([]abomInsightSection, 0, 6)

	newItems := []abomInsightItem{}
	newHigh := []abomInsightItem{}
	highConfItems := []abomInsightItem{}
	lowConfItems := []abomInsightItem{}
	widespreadItems := []abomInsightItem{}
	staleItems := []abomInsightItem{}

	for _, item := range items {
		entry := abomInsightItem{
			ID:          item.ID,
			Category:    item.Category,
			Product:     item.Product,
			Confidence:  item.Confidence,
			DeviceIDs:   append([]string(nil), item.DeviceIDs...),
			DeviceCount: len(item.DeviceIDs),
			FirstSeenMS: item.FirstSeenMS,
			LastSeenMS:  item.LastSeenMS,
		}
		entry.HighAttention = anyDeviceInSet(item.DeviceIDs, highSet)

		if item.FirstSeenMS > 0 && item.FirstSeenMS >= cutoffNew {
			cp := entry
			cp.Reason = fmt.Sprintf("First seen %s — within the last %s.", relativeMS(opts.Now, item.FirstSeenMS), formatDurationMS(opts.WindowMS))
			newItems = append(newItems, cp)
			if cp.HighAttention {
				cpHA := cp
				cpHA.Reason = "Newly observed on an endpoint with open findings — review before drift becomes the norm."
				newHigh = append(newHigh, cpHA)
			}
		}

		switch item.Confidence {
		case abomConfidenceHigh:
			cp := entry
			cp.Reason = fmt.Sprintf("High-confidence ABOM row — %d evidence ref(s) across %d device(s).", len(item.EvidenceRefs), len(item.DeviceIDs))
			highConfItems = append(highConfItems, cp)
		case abomConfidenceLow:
			cp := entry
			cp.Reason = "Low confidence — needs additional evidence before treating as inventory truth."
			lowConfItems = append(lowConfItems, cp)
		}

		if len(item.DeviceIDs) >= widespreadThreshold {
			cp := entry
			cp.Reason = fmt.Sprintf("Observed on %d device(s) — widespread across the fleet.", len(item.DeviceIDs))
			widespreadItems = append(widespreadItems, cp)
		}

		if item.LastSeenMS > 0 && item.LastSeenMS <= cutoffStale {
			cp := entry
			cp.Reason = fmt.Sprintf("Last seen %s — older than %s.", relativeMS(opts.Now, item.LastSeenMS), formatDurationMS(opts.StaleMS))
			staleItems = append(staleItems, cp)
		}
	}

	sections = append(sections, abomInsightSection{
		ID:        abomInsightNew,
		Title:     "New since last review",
		Detail:    "AI capabilities first observed in the selected window.",
		Threshold: fmt.Sprintf("first_seen within last %s", formatDurationMS(opts.WindowMS)),
		Items:     sortInsightsNewestFirst(newItems),
		Total:     len(newItems),
	})
	sections = append(sections, abomInsightSection{
		ID:        abomInsightNewHighAttention,
		Title:     "New on endpoints with open findings",
		Detail:    "Newly observed AI capability on an endpoint that already has investigation activity. Treat as priority.",
		Threshold: fmt.Sprintf("first_seen within last %s AND device has open finding", formatDurationMS(opts.WindowMS)),
		Items:     sortInsightsNewestFirst(newHigh),
		Total:     len(newHigh),
	})
	sections = append(sections, abomInsightSection{
		ID:        abomInsightWidespread,
		Title:     "Widespread capabilities",
		Detail:    "AI capabilities observed across many endpoints. Decide whether they should be sanctioned, scoped, or restricted.",
		Threshold: fmt.Sprintf(">= %d devices", widespreadThreshold),
		Items:     sortInsightsByDeviceCount(widespreadItems),
		Total:     len(widespreadItems),
	})
	sections = append(sections, abomInsightSection{
		ID:        abomInsightHighConfidence,
		Title:     "High-confidence inventory",
		Detail:    "Items where the platform has enough evidence to trust the row without manual review.",
		Threshold: "confidence == high",
		Items:     sortInsightsByDeviceCount(highConfItems),
		Total:     len(highConfItems),
	})
	sections = append(sections, abomInsightSection{
		ID:        abomInsightLowConfidenceReview,
		Title:     "Needs review",
		Detail:    "Items the platform is not yet confident about. Confirm or refute before they become inventory truth.",
		Threshold: "confidence == low",
		Items:     sortInsightsByDeviceCount(lowConfItems),
		Total:     len(lowConfItems),
	})
	sections = append(sections, abomInsightSection{
		ID:        abomInsightStale,
		Title:     "Stale capabilities",
		Detail:    "Items that have not been observed recently. Confirm whether the tool was uninstalled or the agent stopped reporting.",
		Threshold: fmt.Sprintf("last_seen older than %s", formatDurationMS(opts.StaleMS)),
		Items:     sortInsightsByLastSeen(staleItems),
		Total:     len(staleItems),
	})

	return abomInsightsBundle{
		HighAttentionDevices: keysSlice(highSet),
		Sections:             sections,
		Hotspots:             buildABOMHotspots(items, highSet, opts.Now),
	}
}

func buildABOMHotspots(items []abomItem, highSet map[string]struct{}, now int64) []abomEndpointHotspot {
	if len(items) == 0 {
		return []abomEndpointHotspot{}
	}
	type acc struct {
		highConf int
		lowConf  int
		categories map[string]struct{}
		lastSeen   int64
		count      int
	}
	by := map[string]*acc{}
	for _, item := range items {
		for _, dev := range item.DeviceIDs {
			if dev == "" {
				continue
			}
			rec, ok := by[dev]
			if !ok {
				rec = &acc{categories: map[string]struct{}{}}
				by[dev] = rec
			}
			rec.count++
			rec.categories[item.Category] = struct{}{}
			switch item.Confidence {
			case abomConfidenceHigh:
				rec.highConf++
			case abomConfidenceLow:
				rec.lowConf++
			}
			if item.LastSeenMS > rec.lastSeen {
				rec.lastSeen = item.LastSeenMS
			}
		}
	}

	out := make([]abomEndpointHotspot, 0, len(by))
	for dev, rec := range by {
		categories := make([]string, 0, len(rec.categories))
		for c := range rec.categories {
			categories = append(categories, c)
		}
		sort.Strings(categories)
		_, ha := highSet[dev]
		hotspot := abomEndpointHotspot{
			DeviceID:       dev,
			ItemCount:      rec.count,
			HighConfidence: rec.highConf,
			LowConfidence:  rec.lowConf,
			Categories:     categories,
			LastSeenMS:     rec.lastSeen,
			HighAttention:  ha,
			Reason:         hotspotReason(rec.count, rec.lowConf, ha),
		}
		out = append(out, hotspot)
	}
	sort.SliceStable(out, func(i, j int) bool {
		// Hotspots: high attention first, then by item count, then by low-confidence count.
		if out[i].HighAttention != out[j].HighAttention {
			return out[i].HighAttention
		}
		if out[i].ItemCount != out[j].ItemCount {
			return out[i].ItemCount > out[j].ItemCount
		}
		if out[i].LowConfidence != out[j].LowConfidence {
			return out[i].LowConfidence > out[j].LowConfidence
		}
		return out[i].LastSeenMS > out[j].LastSeenMS
	})
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func hotspotReason(count, lowConfidence int, highAttention bool) string {
	if highAttention {
		return fmt.Sprintf("High-attention endpoint (open finding) with %d ABOM rows; %d still need review.", count, lowConfidence)
	}
	if lowConfidence >= 2 {
		return fmt.Sprintf("Endpoint has %d ABOM rows but %d need review — confirm before trusting the inventory.", count, lowConfidence)
	}
	return fmt.Sprintf("Dense AI footprint — %d ABOM rows observed on this endpoint.", count)
}

func highAttentionDeviceIDs(findings []findingRecord) []string {
	set := make(map[string]struct{}, len(findings))
	for _, f := range findings {
		if f.DeviceID == "" {
			continue
		}
		set[f.DeviceID] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func anyDeviceInSet(devices []string, set map[string]struct{}) bool {
	for _, d := range devices {
		if _, ok := set[d]; ok {
			return true
		}
	}
	return false
}

func sortInsightsNewestFirst(rows []abomInsightItem) []abomInsightItem {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].FirstSeenMS > rows[j].FirstSeenMS })
	if len(rows) > 24 {
		rows = rows[:24]
	}
	return rows
}

func sortInsightsByDeviceCount(rows []abomInsightItem) []abomInsightItem {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].DeviceCount != rows[j].DeviceCount {
			return rows[i].DeviceCount > rows[j].DeviceCount
		}
		return rows[i].LastSeenMS > rows[j].LastSeenMS
	})
	if len(rows) > 24 {
		rows = rows[:24]
	}
	return rows
}

func sortInsightsByLastSeen(rows []abomInsightItem) []abomInsightItem {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].LastSeenMS < rows[j].LastSeenMS })
	if len(rows) > 24 {
		rows = rows[:24]
	}
	return rows
}

func parsePositiveDuration(raw string, fallback int64) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func relativeMS(now, ts int64) string {
	if ts <= 0 || now < ts {
		return "moments ago"
	}
	delta := now - ts
	switch {
	case delta < 60*1000:
		return fmt.Sprintf("%ds ago", delta/1000)
	case delta < 60*60*1000:
		return fmt.Sprintf("%dm ago", delta/(60*1000))
	case delta < 24*60*60*1000:
		return fmt.Sprintf("%dh ago", delta/(60*60*1000))
	default:
		return fmt.Sprintf("%dd ago", delta/(24*60*60*1000))
	}
}

func formatDurationMS(ms int64) string {
	switch {
	case ms < 60*1000:
		return fmt.Sprintf("%ds", ms/1000)
	case ms < 60*60*1000:
		return fmt.Sprintf("%dm", ms/(60*1000))
	case ms < 24*60*60*1000:
		return fmt.Sprintf("%dh", ms/(60*60*1000))
	default:
		return fmt.Sprintf("%dd", ms/(24*60*60*1000))
	}
}

func keysSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

