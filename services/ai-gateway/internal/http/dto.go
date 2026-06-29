package httpapi

import (
	"encoding/json"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

type modelProfileResponse struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Purpose           service.Purpose  `json:"purpose"`
	Provider          service.Provider `json:"provider"`
	BaseURL           string           `json:"baseUrl"`
	Model             string           `json:"model"`
	Enabled           bool             `json:"enabled"`
	IsDefault         bool             `json:"isDefault"`
	TimeoutMS         int              `json:"timeoutMs"`
	APIKeyConfigured  bool             `json:"apiKeyConfigured"`
	SupportsStreaming bool             `json:"supportsStreaming"`
	Dimensions        *int             `json:"dimensions"`
	TopN              *int             `json:"topN"`
	DefaultParameters json.RawMessage  `json:"defaultParameters"`
	CreatedAt         time.Time        `json:"createdAt"`
	UpdatedAt         time.Time        `json:"updatedAt"`
}

type createModelProfileRequest struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Purpose           service.Purpose  `json:"purpose"`
	Provider          service.Provider `json:"provider"`
	BaseURL           string           `json:"baseUrl"`
	Model             string           `json:"model"`
	APIKey            string           `json:"apiKey"`
	Enabled           *bool            `json:"enabled"`
	IsDefault         *bool            `json:"isDefault"`
	TimeoutMS         *int             `json:"timeoutMs"`
	SupportsStreaming *bool            `json:"supportsStreaming"`
	Dimensions        *int             `json:"dimensions"`
	TopN              *int             `json:"topN"`
	DefaultParameters json.RawMessage  `json:"defaultParameters"`
}

type updateModelProfileRequest struct {
	Name              *string           `json:"name"`
	Provider          *service.Provider `json:"provider"`
	BaseURL           *string           `json:"baseUrl"`
	Model             *string           `json:"model"`
	APIKey            *string           `json:"apiKey"`
	Enabled           *bool             `json:"enabled"`
	IsDefault         *bool             `json:"isDefault"`
	TimeoutMS         *int              `json:"timeoutMs"`
	SupportsStreaming *bool             `json:"supportsStreaming"`
	Dimensions        *int              `json:"dimensions"`
	TopN              *int              `json:"topN"`
	DefaultParameters *json.RawMessage  `json:"defaultParameters"`
}

func profilesFromDomain(items []service.ModelProfile) []modelProfileResponse {
	out := make([]modelProfileResponse, len(items))
	for i, item := range items {
		out[i] = profileFromDomain(item)
	}
	return out
}

func profileFromDomain(profile service.ModelProfile) modelProfileResponse {
	defaultParameters := profile.DefaultParameters
	if len(defaultParameters) == 0 {
		defaultParameters = json.RawMessage(`{}`)
	}
	return modelProfileResponse{
		ID:                profile.ID,
		Name:              profile.Name,
		Purpose:           profile.Purpose,
		Provider:          profile.Provider,
		BaseURL:           profile.BaseURL,
		Model:             profile.Model,
		Enabled:           profile.Enabled,
		IsDefault:         profile.IsDefault,
		TimeoutMS:         profile.TimeoutMS,
		APIKeyConfigured:  profile.APIKeyConfigured,
		SupportsStreaming: profile.SupportsStreaming,
		Dimensions:        cloneIntPtr(profile.Dimensions),
		TopN:              cloneIntPtr(profile.TopN),
		DefaultParameters: append(json.RawMessage(nil), defaultParameters...),
		CreatedAt:         profile.CreatedAt,
		UpdatedAt:         profile.UpdatedAt,
	}
}

func createInputFromRequest(payload createModelProfileRequest) service.CreateModelProfileInput {
	return service.CreateModelProfileInput{
		ID:                payload.ID,
		Name:              payload.Name,
		Purpose:           payload.Purpose,
		Provider:          payload.Provider,
		BaseURL:           payload.BaseURL,
		Model:             payload.Model,
		APIKey:            payload.APIKey,
		Enabled:           payload.Enabled,
		IsDefault:         payload.IsDefault,
		TimeoutMS:         payload.TimeoutMS,
		SupportsStreaming: payload.SupportsStreaming,
		Dimensions:        payload.Dimensions,
		TopN:              payload.TopN,
		DefaultParameters: payload.DefaultParameters,
	}
}

func updateInputFromRequest(id string, payload updateModelProfileRequest) service.UpdateModelProfileInput {
	return service.UpdateModelProfileInput{
		ID:                id,
		Name:              payload.Name,
		Provider:          payload.Provider,
		BaseURL:           payload.BaseURL,
		Model:             payload.Model,
		APIKey:            payload.APIKey,
		Enabled:           payload.Enabled,
		IsDefault:         payload.IsDefault,
		TimeoutMS:         payload.TimeoutMS,
		SupportsStreaming: payload.SupportsStreaming,
		Dimensions:        payload.Dimensions,
		TopN:              payload.TopN,
		DefaultParameters: payload.DefaultParameters,
	}
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
