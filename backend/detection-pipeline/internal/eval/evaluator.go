package eval

import (
	"encoding/json"
	"fmt"
	"strings"

	"aegisflux/backend/detection-pipeline/internal/ingestclient"
)

// Batch indexes visibility rows for rule evaluation (lab validation; best-effort correlation).
type Batch struct {
	ProcessesByGUID map[string]processPayload
	DNS             []dnsPayload
	Flows           []flowPayload
	SASE            []sasePayload
	Extensions      []extensionPayload
}

type processPayload struct {
	Name              string
	CommandLine       string
	Path              string
	ParentProcessGUID string
}

type dnsPayload struct {
	Query string
}

type flowPayload struct {
	ProcessGUID    string
	RemotePort     *int
	RemoteHostname string
}

type sasePayload struct {
	Vendor  string
	Product string
	Name    string
	Source  string
}

type extensionPayload struct {
	ExtensionID string
	Name        string
	Permissions []string
}

// BuildBatch parses ingest visibility events into a batch index.
func BuildBatch(events []ingestclient.VisibilityEvent) (*Batch, error) {
	b := &Batch{
		ProcessesByGUID: make(map[string]processPayload),
	}
	for _, ev := range events {
		switch ev.EventType {
		case "aegis.process.started":
			var p struct {
				ProcessGUID       string  `json:"process_guid"`
				ParentProcessGUID *string `json:"parent_process_guid"`
				Name              string  `json:"name"`
				Path              *string `json:"path"`
				CommandLine       *string `json:"command_line"`
			}
			if err := json.Unmarshal(ev.Payload, &p); err != nil {
				continue
			}
			if p.ProcessGUID == "" {
				continue
			}
			pp := processPayload{Name: p.Name}
			if p.Path != nil {
				pp.Path = *p.Path
			}
			if p.CommandLine != nil {
				pp.CommandLine = *p.CommandLine
			}
			if p.ParentProcessGUID != nil {
				pp.ParentProcessGUID = *p.ParentProcessGUID
			}
			b.ProcessesByGUID[p.ProcessGUID] = pp
		case "aegis.dns.observed":
			var d struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(ev.Payload, &d); err != nil {
				continue
			}
			if d.Query != "" {
				b.DNS = append(b.DNS, dnsPayload{Query: d.Query})
			}
		case "aegis.flow.started":
			var f struct {
				ProcessGUID    *string `json:"process_guid"`
				RemotePort     *int    `json:"remote_port"`
				RemoteHostname *string `json:"remote_hostname"`
			}
			if err := json.Unmarshal(ev.Payload, &f); err != nil {
				continue
			}
			fp := flowPayload{}
			if f.ProcessGUID != nil {
				fp.ProcessGUID = *f.ProcessGUID
			}
			fp.RemotePort = f.RemotePort
			if f.RemoteHostname != nil {
				fp.RemoteHostname = *f.RemoteHostname
			}
			b.Flows = append(b.Flows, fp)
		case "aegis.sase_component.observed":
			var s struct {
				Vendor  string `json:"vendor"`
				Product string `json:"product"`
				Name    string `json:"name"`
				Source  string `json:"source"`
			}
			if err := json.Unmarshal(ev.Payload, &s); err != nil {
				continue
			}
			b.SASE = append(b.SASE, sasePayload{Vendor: s.Vendor, Product: s.Product, Name: s.Name, Source: s.Source})
		case "aegis.browser_extension.observed":
			var e struct {
				ExtensionID string   `json:"extension_id"`
				Name        string   `json:"name"`
				Permissions []string `json:"permissions"`
			}
			if err := json.Unmarshal(ev.Payload, &e); err != nil {
				continue
			}
			b.Extensions = append(b.Extensions, extensionPayload{
				ExtensionID: e.ExtensionID,
				Name:        e.Name,
				Permissions: e.Permissions,
			})
		}
	}
	return b, nil
}

// RuleMatches returns true if the rule's match clause is satisfied by the batch.
func RuleMatches(rule map[string]any, b *Batch) (bool, error) {
	raw, ok := rule["match"]
	if !ok {
		return false, fmt.Errorf("rule missing match")
	}
	clause, ok := raw.(map[string]any)
	if !ok {
		return false, fmt.Errorf("match not object")
	}
	return evalClause(clause, b)
}

func evalClause(c map[string]any, b *Batch) (bool, error) {
	if _, ok := c["op"]; ok {
		return evalGroup(c, b)
	}
	if _, ok := c["process"]; ok {
		return evalProcessLeaf(c, b)
	}
	if _, ok := c["dns"]; ok {
		return evalDNSLeaf(c, b)
	}
	if _, ok := c["flow"]; ok {
		return evalFlowLeaf(c, b)
	}
	if _, ok := c["sase_component"]; ok {
		return evalSASELeaf(c, b)
	}
	if _, ok := c["browser_extension"]; ok {
		return evalExtLeaf(c, b)
	}
	return false, fmt.Errorf("unknown clause keys %v", keysSample(c))
}

func evalGroup(c map[string]any, b *Batch) (bool, error) {
	op, _ := c["op"].(string)
	rawOf, ok := c["of"].([]any)
	if !ok {
		return false, fmt.Errorf("group missing of")
	}
	clauses := make([]map[string]any, 0, len(rawOf))
	for _, x := range rawOf {
		m, ok := x.(map[string]any)
		if !ok {
			return false, fmt.Errorf("group child not object")
		}
		clauses = append(clauses, m)
	}
	switch op {
	case "all_of":
		for _, ch := range clauses {
			ok, err := evalClause(ch, b)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	case "any_of":
		min := 1
		if v, ok := c["min_match"].(float64); ok {
			min = int(v)
		}
		if min < 1 {
			min = 1
		}
		n := 0
		for _, ch := range clauses {
			ok, err := evalClause(ch, b)
			if err != nil {
				return false, err
			}
			if ok {
				n++
			}
		}
		return n >= min, nil
	default:
		return false, fmt.Errorf("unknown op %q", op)
	}
}

func evalProcessLeaf(c map[string]any, b *Batch) (bool, error) {
	pm, ok := c["process"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("process leaf malformed")
	}
	ci := true
	if v, ok := pm["case_insensitive"].(bool); ok {
		ci = v
	}
	for _, proc := range b.ProcessesByGUID {
		if matchProcess(b, pm, proc, ci) {
			return true, nil
		}
	}
	return false, nil
}

func matchProcess(b *Batch, pm map[string]any, proc processPayload, ci bool) bool {
	name := proc.Name
	cmd := proc.CommandLine
	path := proc.Path
	parentName := ""
	if proc.ParentProcessGUID != "" {
		if par, ok := b.ProcessesByGUID[proc.ParentProcessGUID]; ok {
			parentName = par.Name
		}
	}
	if ci {
		name = strings.ToLower(name)
		cmd = strings.ToLower(cmd)
		path = strings.ToLower(path)
		parentName = strings.ToLower(parentName)
	}
	if arr, ok := pm["executable_names_any"].([]any); ok {
		if !containsStringAny(name, arr, ci) {
			return false
		}
	}
	if arr, ok := pm["executable_name_contains_any"].([]any); ok {
		if !substringAny(name, arr, ci) {
			return false
		}
	}
	if arr, ok := pm["command_line_contains_any"].([]any); ok {
		if !substringAny(cmd, arr, ci) {
			return false
		}
	}
	if arr, ok := pm["parent_executable_names_any"].([]any); ok {
		if !containsStringAny(parentName, arr, ci) {
			return false
		}
	}
	if arr, ok := pm["process_path_contains_any"].([]any); ok {
		if !substringAny(path, arr, ci) {
			return false
		}
	}
	return true
}

func evalDNSLeaf(c map[string]any, b *Batch) (bool, error) {
	dm, ok := c["dns"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("dns leaf malformed")
	}
	arr, ok := dm["query_contains_any"].([]any)
	if !ok {
		return false, nil
	}
	for _, d := range b.DNS {
		q := strings.ToLower(d.Query)
		if substringAny(q, arr, true) {
			return true, nil
		}
	}
	return false, nil
}

func evalFlowLeaf(c map[string]any, b *Batch) (bool, error) {
	fm, ok := c["flow"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("flow leaf malformed")
	}
	if v, ok := fm["has_any_flow"].(bool); ok && v {
		return len(b.Flows) > 0, nil
	}
	for _, fl := range b.Flows {
		if matchFlow(fm, fl) {
			return true, nil
		}
	}
	return false, nil
}

func matchFlow(fm map[string]any, fl flowPayload) bool {
	if arr, ok := fm["remote_ports_any"].([]any); ok && len(arr) > 0 {
		if fl.RemotePort == nil {
			return false
		}
		found := false
		for _, x := range arr {
			p, ok := numberToInt(x)
			if ok && p == *fl.RemotePort {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if arr, ok := fm["remote_host_contains_any"].([]any); ok {
		h := strings.ToLower(fl.RemoteHostname)
		if !substringAny(h, arr, true) {
			return false
		}
	}
	return true
}

func evalSASELeaf(c map[string]any, b *Batch) (bool, error) {
	sm, ok := c["sase_component"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("sase leaf malformed")
	}
	for _, s := range b.SASE {
		hay := strings.ToLower(s.Vendor + " " + s.Product + " " + s.Name + " " + s.Source)
		if arr, ok := sm["display_name_contains_any"].([]any); ok {
			if substringAny(hay, arr, true) {
				return true, nil
			}
		}
		if arr, ok := sm["vendor_contains_any"].([]any); ok {
			if substringAny(strings.ToLower(s.Vendor), arr, true) {
				return true, nil
			}
		}
	}
	return false, nil
}

func evalExtLeaf(c map[string]any, b *Batch) (bool, error) {
	em, ok := c["browser_extension"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("browser_extension leaf malformed")
	}
	for _, e := range b.Extensions {
		id := strings.ToLower(e.ExtensionID)
		n := strings.ToLower(e.Name)
		perm := strings.ToLower(strings.Join(e.Permissions, " "))
		if arr, ok := em["extension_id_contains_any"].([]any); ok {
			if substringAny(id, arr, true) {
				return true, nil
			}
		}
		if arr, ok := em["name_contains_any"].([]any); ok {
			if substringAny(n, arr, true) {
				return true, nil
			}
		}
		if arr, ok := em["permissions_contains_any"].([]any); ok {
			if substringAny(perm, arr, true) {
				return true, nil
			}
		}
	}
	return false, nil
}

func substringAny(s string, needles []any, lowerNeedles bool) bool {
	for _, n := range needles {
		ns, ok := n.(string)
		if !ok {
			continue
		}
		if lowerNeedles {
			ns = strings.ToLower(ns)
		}
		if ns != "" && strings.Contains(s, ns) {
			return true
		}
	}
	return false
}

func containsStringAny(s string, names []any, ci bool) bool {
	for _, n := range names {
		ns, ok := n.(string)
		if !ok {
			continue
		}
		a, b := s, ns
		if ci {
			a, b = strings.ToLower(a), strings.ToLower(b)
		}
		if a == b {
			return true
		}
	}
	return false
}

func numberToInt(x any) (int, bool) {
	switch v := x.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

func keysSample(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
		if len(out) >= 8 {
			break
		}
	}
	return out
}
