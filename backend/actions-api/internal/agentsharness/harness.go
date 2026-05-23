package agentsharness

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DataSource supplies read-only platform views and redaction for the harness.
// Implementations must be safe to call from the API goroutine handling the request.
type DataSource interface {
	AllowExternalAI() bool
	DefaultProviderKind() string
	RedactJSONPreview(v any) string
	DeviceEvidenceSummary(deviceID string) map[string]any
	FindingsEvidencePaths(deviceID, findingID string) []map[string]string
	DetectionCandidateLookup(candidateID, deviceID string) map[string]any
}

// Harness executes governed agent jobs with typed, audited tool calls.
type Harness struct {
	Data DataSource
}

func toolsInOrder(agentID string) []string {
	switch agentID {
	case AgentEndpointAnalyst:
		return []string{ToolDeviceEvidenceSummary, ToolFindingsEvidencePaths, ToolDetectionCandidateLookup}
	case AgentDetectionResearcher:
		return []string{ToolFindingsEvidencePaths, ToolDetectionCandidateLookup}
	case AgentPackAuthor:
		return []string{ToolDetectionCandidateLookup}
	case AgentSimulationAgent:
		return []string{ToolDeviceEvidenceSummary, ToolFindingsEvidencePaths}
	case AgentControlDesigner:
		return []string{ToolFindingsEvidencePaths, ToolDeviceEvidenceSummary}
	case AgentGovernanceReviewer:
		return []string{ToolDeviceEvidenceSummary, ToolDetectionCandidateLookup}
	default:
		return nil
	}
}

// Run executes a synchronous lab job: queued → running → terminal status.
func (h *Harness) Run(ctx context.Context, spec RunSpec) (*JobRecord, *RunRecord, error) {
	if h == nil || h.Data == nil {
		return nil, nil, fmt.Errorf("harness not configured")
	}
	if _, ok := agentMeta(spec.AgentID); !ok {
		return nil, nil, fmt.Errorf("unknown agent: %s", spec.AgentID)
	}
	if spec.DeviceID == "" {
		return nil, nil, fmt.Errorf("device_id required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	_ = ctx

	now := time.Now().UnixMilli()
	jobID := uuid.NewString()
	runID := uuid.NewString()

	job := &JobRecord{
		JobID:     jobID,
		AgentID:   spec.AgentID,
		DeviceID:  spec.DeviceID,
		FindingID: spec.FindingID,
		Status:    JobQueued,
		CreatedMS: now,
		UpdatedMS: now,
	}
	job.Status = JobRunning
	job.UpdatedMS = time.Now().UnixMilli()

	run := &RunRecord{
		RunID:          runID,
		JobID:          jobID,
		AgentID:        spec.AgentID,
		DeviceID:       spec.DeviceID,
		FindingID:      spec.FindingID,
		ToolCalls:      []ToolCallRecord{},
		Status:         JobRunning,
		StartedMS:      time.Now().UnixMilli(),
		PrivacyApplied: true,
	}

	previewPayload := map[string]any{
		"device_id": spec.DeviceID,
		"context":   spec.Context,
	}
	if spec.FindingID != "" {
		previewPayload["finding_id"] = spec.FindingID
	}
	run.PromptRedactedPreview = h.Data.RedactJSONPreview(previewPayload)

	allowExt := h.Data.AllowExternalAI()
	run.ProviderKind = h.Data.DefaultProviderKind()
	if allowExt {
		run.Model = "external_stub"
	} else {
		run.Model = "lab/deterministic"
	}

	order := toolsInOrder(spec.AgentID)
	if len(order) == 0 {
		run.Status = JobFailed
		run.Error = "no tool plan for agent"
		end := time.Now().UnixMilli()
		run.EndedMS = end
		run.DurationMS = end - run.StartedMS
		job.Status = JobFailed
		job.Error = run.Error
		job.UpdatedMS = end
		return job, run, nil
	}

	for _, toolID := range order {
		if !toolAllowedForAgent(spec.AgentID, toolID) {
			run.Status = JobFailed
			run.Error = fmt.Sprintf("internal allowlist mismatch for tool %s", toolID)
			break
		}
		rec := h.invokeTool(toolID, spec)
		run.ToolCalls = append(run.ToolCalls, rec)
		if rec.Error != "" {
			run.Status = JobFailed
			run.Error = rec.Error
			job.Status = JobFailed
			job.Error = rec.Error
			break
		}
	}

	if run.Status != JobFailed {
		assessment, evidence, confidence, next := synthesizeEndpointStyle(allowExt, run.PromptRedactedPreview)
		run.Assessment = assessment
		run.EvidenceSummary = evidence
		run.Confidence = confidence
		run.RecommendedNextAction = next

		impacting := productImpacting(spec)
		concl := BuildEvidenceBoundConclusion(spec, run.ToolCalls, assessment, evidence, confidence, next, allowExt)
		errs := ValidateEvidenceBoundConclusion(concl, ValidateEvidenceOptions{ProductImpacting: impacting})
		if len(errs) > 0 {
			run.Status = JobFailed
			job.Status = JobFailed
			run.EvidenceBoundValidationErrors = errs
			run.Error = "evidence_bound_validation_failed: " + strings.Join(errs, "; ")
			job.Error = run.Error
			run.EvidenceBoundConclusion = concl
		} else {
			run.Status = JobCompleted
			job.Status = JobCompleted
			run.EvidenceBoundConclusion = concl
		}
	}

	end := time.Now().UnixMilli()
	run.EndedMS = end
	run.DurationMS = end - run.StartedMS
	job.UpdatedMS = end

	return job, run, nil
}

func (h *Harness) invokeTool(toolID string, spec RunSpec) ToolCallRecord {
	callID := uuid.NewString()
	start := time.Now().UnixMilli()
	var out map[string]any
	var in map[string]any

	switch toolID {
	case ToolDeviceEvidenceSummary:
		in = map[string]any{"device_id": spec.DeviceID}
		out = h.Data.DeviceEvidenceSummary(spec.DeviceID)
	case ToolFindingsEvidencePaths:
		in = map[string]any{"device_id": spec.DeviceID, "finding_id": spec.FindingID}
		paths := h.Data.FindingsEvidencePaths(spec.DeviceID, spec.FindingID)
		out = map[string]any{"paths": paths}
	case ToolDetectionCandidateLookup:
		in = map[string]any{"device_id": spec.DeviceID, "candidate_id": spec.CandidateID}
		out = h.Data.DetectionCandidateLookup(spec.CandidateID, spec.DeviceID)
	default:
		end := time.Now().UnixMilli()
		return ToolCallRecord{
			CallID:     callID,
			ToolID:     toolID,
			InputJSON:  mustJSON(in),
			StartedMS:  start,
			EndedMS:    end,
			DurationMS: end - start,
			Error:      "unknown tool",
		}
	}

	inb := mustJSON(in)
	outb := mustJSON(out)
	end := time.Now().UnixMilli()
	return ToolCallRecord{
		CallID:     callID,
		ToolID:     toolID,
		InputJSON:  inb,
		OutputJSON: outb,
		StartedMS:  start,
		EndedMS:    end,
		DurationMS: end - start,
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

func synthesizeEndpointStyle(allowExternal bool, redactedPreview string) (assessment, evidence, confidence, next string) {
	if !allowExternal {
		assessment = "External AI calls are disabled; deterministic summary from redacted context."
		confidence = "High (rules-based)"
		next = "Enable governed external AI if you need model narrative."
	} else {
		assessment = "External AI allowed; production would invoke default provider."
		confidence = "Medium (model-dependent)"
		next = "Review audit before sharing externally."
	}
	evidence = redactedPreview
	if len(evidence) > 1200 {
		evidence = evidence[:1200] + "..."
	}
	return assessment, evidence, confidence, next
}

func productImpacting(spec RunSpec) bool {
	if spec.ProductImpacting != nil {
		return *spec.ProductImpacting
	}
	return true
}
