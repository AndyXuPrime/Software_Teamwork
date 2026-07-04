package repository

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestStreamEventSeqInt32RejectsInvalidValues(t *testing.T) {
	if _, err := streamEventSeqInt32(-1); err == nil {
		t.Fatal("expected negative cursor to fail")
	}
	if _, err := streamEventSeqInt32(math.MaxInt32 + 1); err == nil {
		t.Fatal("expected overflow cursor to fail")
	}
	if got, err := streamEventSeqInt32(math.MaxInt32); err != nil || got != math.MaxInt32 {
		t.Fatalf("streamEventSeqInt32(MaxInt32) = %d, %v", got, err)
	}
}

func TestMessageCitationLegacySelectDoesNotRequireSnapshotMigrationColumns(t *testing.T) {
	for _, column := range []string{
		"ci.response_run_id",
		"ci.content_preview",
		"ci.is_source_available",
		"ci.source_unavailable_reason",
	} {
		if strings.Contains(messageCitationLegacySelect, column) {
			t.Fatalf("legacy message citation query should not require migration 0006 column %q: %s", column, messageCitationLegacySelect)
		}
	}
	if strings.Contains(messageCitationLegacySelect, "FALSE AS is_source_available") {
		t.Fatalf("legacy message citation query should not hard-code source availability to false: %s", messageCitationLegacySelect)
	}
}

func TestAgentConfigFromCreateInputPreservesExplicitEmptyToolWhitelist(t *testing.T) {
	config := agentConfigFromCreateInput(service.CreateQAConfigVersionInput{
		Agent:            service.AgentConfig{EnabledToolNames: []string{}},
		EnabledToolNames: []string{"search_knowledge"},
	})

	if config.EnabledToolNames == nil || len(config.EnabledToolNames) != 0 {
		t.Fatalf("enabledToolNames=%#v, want explicit empty whitelist", config.EnabledToolNames)
	}
}

func TestAgentConfigFromCreateInputFallsBackToLegacyToolNamesWhenUnset(t *testing.T) {
	config := agentConfigFromCreateInput(service.CreateQAConfigVersionInput{
		EnabledToolNames: []string{"search_knowledge"},
	})

	if !reflect.DeepEqual(config.EnabledToolNames, []string{"search_knowledge"}) {
		t.Fatalf("enabledToolNames=%#v, want legacy tool names", config.EnabledToolNames)
	}
}

func TestAgentConfigFromCreateInputUsesDefaultToolNamesWhenUnset(t *testing.T) {
	config := agentConfigFromCreateInput(service.CreateQAConfigVersionInput{})

	if !reflect.DeepEqual(config.EnabledToolNames, service.DefaultAgentConfig().EnabledToolNames) {
		t.Fatalf("enabledToolNames=%#v, want defaults", config.EnabledToolNames)
	}
}

func TestRetrievalConfigFromCreateInputPreservesExplicitZeroScoreThreshold(t *testing.T) {
	var input service.CreateQAConfigVersionInput
	if err := json.Unmarshal([]byte(`{"retrieval":{"topK":5,"scoreThreshold":0}}`), &input); err != nil {
		t.Fatal(err)
	}

	config := retrievalConfigFromCreateInput(input)

	if config.ScoreThreshold != 0 {
		t.Fatalf("scoreThreshold=%v, want explicit zero", config.ScoreThreshold)
	}
	if config.TopK != 5 {
		t.Fatalf("topK=%d, want 5", config.TopK)
	}
}

func TestRetrievalConfigFromCreateInputFallsBackToDefaultThresholdWhenUnset(t *testing.T) {
	var input service.CreateQAConfigVersionInput
	if err := json.Unmarshal([]byte(`{"retrieval":{"topK":5}}`), &input); err != nil {
		t.Fatal(err)
	}

	config := retrievalConfigFromCreateInput(input)

	if config.ScoreThreshold != .7 {
		t.Fatalf("scoreThreshold=%v, want default .7", config.ScoreThreshold)
	}
}

func TestRetrievalConfigFromCreateInputUsesLegacySimilarityThreshold(t *testing.T) {
	config := retrievalConfigFromCreateInput(service.CreateQAConfigVersionInput{
		TopK:                6,
		SimilarityThreshold: .35,
	})

	if config.TopK != 6 || config.ScoreThreshold != .35 {
		t.Fatalf("retrieval=%+v, want legacy topK and threshold", config)
	}
}

func TestToolCallAuditSummariesDeriveSourceAndFailure(t *testing.T) {
	if got := toolSourceName("search_knowledge"); got != "qa_builtin" {
		t.Fatalf("builtin source=%q", got)
	}
	if got := toolSourceName("kbserver__search"); got != "kbserver" {
		t.Fatalf("prefixed source=%q", got)
	}

	code, message := toolCallErrorSummary("tool.failed", map[string]any{
		"raw": `{"error":{"code":"retrieval_failed","message":"knowledge retrieval service failed"}}`,
	})
	if code != "retrieval_failed" || message != "knowledge retrieval service failed" {
		t.Fatalf("error summary code=%q message=%q", code, message)
	}
}

func TestToolCallEventPayloadCarriesModelInvocationID(t *testing.T) {
	payload := map[string]any{
		"iterationNo":       1,
		"modelInvocationId": "invocation-id",
		"toolCallId":        "call-1",
		"tool":              "search_knowledge",
	}

	if got, _ := payload["modelInvocationId"].(string); got != "invocation-id" {
		t.Fatalf("modelInvocationId=%q", got)
	}
}

func TestReportArtifactFromJSONKeepsSafeArtifact(t *testing.T) {
	raw := []byte(`{"artifactType":"report_generation","reportId":"rpt-1","jobId":"job-1","jobType":"content_generation","jobStatus":"running","fileStatus":"succeeded","fileSize":2048,"downloadPath":"/api/v1/report-files/file-1/content","detailPath":"/api/v1/reports/rpt-1","ignored":"extra","preview":{"title":"Report generation is running","statusText":"running","ignored":"extra"}}`)

	artifact, ok := reportArtifactFromJSON(raw)

	if !ok {
		t.Fatal("expected safe artifact to be accepted")
	}
	if artifact["reportId"] != "rpt-1" || artifact["jobId"] != "job-1" {
		t.Fatalf("artifact=%+v", artifact)
	}
	if artifact["fileStatus"] != "succeeded" || artifact["downloadPath"] != "/api/v1/report-files/file-1/content" {
		t.Fatalf("artifact paths/status=%+v", artifact)
	}
	if artifact["jobType"] != "content_generation" || artifact["fileSize"] != int64(2048) {
		t.Fatalf("artifact job type/file size=%+v", artifact)
	}
	if _, ok := artifact["ignored"]; ok {
		t.Fatalf("unexpected extra artifact field: %+v", artifact)
	}
	preview, _ := artifact["preview"].(map[string]any)
	if _, ok := preview["ignored"]; ok {
		t.Fatalf("unexpected extra preview field: %+v", preview)
	}
}

func TestReportArtifactFromJSONRejectsUnsafeArtifact(t *testing.T) {
	raw := []byte(`{"artifactType":"report_generation","reportId":"rpt-1","preview":{"summary":"internal URL http://localhost:9000/private"}}`)

	if artifact, ok := reportArtifactFromJSON(raw); ok {
		t.Fatalf("unsafe artifact accepted: %+v", artifact)
	}
}

func TestReportArtifactFromJSONDropsInvalidPublicContractValues(t *testing.T) {
	raw := []byte(`{"artifactType":"report_generation","reportId":"rpt-1","jobType":"unknown_job","jobStatus":"weird","fileStatus":"completed","format":"pdf","fileSize":12.5,"downloadPath":"/api/v1/files/file-1","detailPath":"/api/v1/reports/rpt-1/extra"}`)

	artifact, ok := reportArtifactFromJSON(raw)

	if !ok {
		t.Fatal("expected artifact with valid identity to be accepted")
	}
	for _, key := range []string{"jobType", "jobStatus", "fileStatus", "format", "fileSize", "downloadPath", "detailPath"} {
		if _, ok := artifact[key]; ok {
			t.Fatalf("invalid contract field %q was kept: %+v", key, artifact)
		}
	}
}

func TestReportArtifactFromJSONDropsNegativeFileSize(t *testing.T) {
	raw := []byte(`{"artifactType":"report_generation","reportId":"rpt-1","fileSize":-1}`)

	artifact, ok := reportArtifactFromJSON(raw)

	if !ok {
		t.Fatal("expected artifact with valid identity to be accepted")
	}
	if _, ok := artifact["fileSize"]; ok {
		t.Fatalf("negative fileSize was kept: %+v", artifact)
	}
}

func TestMergeReportArtifactUpdatesStableArtifact(t *testing.T) {
	existing := []service.ReportArtifact{{
		"artifactType": "report_generation",
		"reportId":     "rpt-1",
		"jobId":        "job-1",
		"jobStatus":    "running",
	}}
	updated := service.ReportArtifact{
		"artifactType": "report_generation",
		"reportId":     "rpt-1",
		"jobId":        "job-1",
		"jobStatus":    "succeeded",
		"reportFileId": "file-1",
		"downloadPath": "/api/v1/report-files/file-1/content",
		"preview":      map[string]any{"title": "DOCX report export ready"},
	}

	merged := mergeReportArtifact(existing, updated)

	if len(merged) != 1 || merged[0]["jobStatus"] != "succeeded" || merged[0]["reportFileId"] != "file-1" {
		t.Fatalf("merged=%+v", merged)
	}
}

func TestMarshalCitationMetadataNormalizesAttachmentID(t *testing.T) {
	raw, err := marshalCitationMetadata(service.Citation{
		AttachmentID: " attachment-canonical ",
		Metadata: map[string]any{
			"attachmentId":  "spoofed",
			"attachment_id": "legacy",
			"keep":          "value",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["attachmentId"] != "attachment-canonical" {
		t.Fatalf("attachmentId=%v, want canonical attachment id", got["attachmentId"])
	}
	if _, ok := got["attachment_id"]; ok {
		t.Fatalf("attachment_id should be removed: %v", got)
	}
	if got["keep"] != "value" {
		t.Fatalf("metadata keep=%v, want value", got["keep"])
	}
}

func TestMarshalCitationMetadataHandlesNilMetadata(t *testing.T) {
	raw, err := marshalCitationMetadata(service.Citation{})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{}" {
		t.Fatalf("metadata json=%s, want {}", raw)
	}
}
