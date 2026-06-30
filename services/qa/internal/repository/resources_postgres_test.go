package repository

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestApplyQAConfigVersionCompatibilityFieldsMirrorsAgentConfig(t *testing.T) {
	config := service.QAConfigVersion{
		Agent: service.AgentConfig{
			MaxIterations:         6,
			ToolTimeoutSeconds:    11,
			ModelTimeoutSeconds:   61,
			OverallTimeoutSeconds: 121,
			EnabledToolNames:      []string{"search_knowledge", "general_chat"},
		},
	}

	applyQAConfigVersionCompatibilityFields(&config)

	if config.MaxIterations != config.Agent.MaxIterations ||
		config.ToolTimeoutSeconds != config.Agent.ToolTimeoutSeconds ||
		config.ModelTimeoutSeconds != config.Agent.ModelTimeoutSeconds ||
		config.OverallTimeoutSeconds != config.Agent.OverallTimeoutSeconds {
		t.Fatalf("flat fields were not mirrored from agent: %+v", config)
	}
	if !reflect.DeepEqual(config.EnabledToolNames, config.Agent.EnabledToolNames) {
		t.Fatalf("enabledToolNames = %#v, want %#v", config.EnabledToolNames, config.Agent.EnabledToolNames)
	}
	config.Agent.EnabledToolNames[0] = "mutated"
	if config.EnabledToolNames[0] != "search_knowledge" {
		t.Fatalf("enabledToolNames aliases agent slice: %#v", config.EnabledToolNames)
	}
}

func TestMessageCitationSelectDoesNotRequireSnapshotMigrationColumns(t *testing.T) {
	for _, column := range []string{
		"ci.response_run_id",
		"ci.content_preview",
		"ci.is_source_available",
		"ci.source_unavailable_reason",
	} {
		if strings.Contains(messageCitationSelect, column) {
			t.Fatalf("message citation query should not require migration 0006 column %q: %s", column, messageCitationSelect)
		}
	}
}
