package adapter

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

type createParserConfigRequest struct {
	Name                  string                `json:"name"`
	Backend               service.ParserBackend `json:"backend"`
	Enabled               *bool                 `json:"enabled"`
	IsDefault             *bool                 `json:"isDefault"`
	Concurrency           int                   `json:"concurrency"`
	SupportedContentTypes []string              `json:"supportedContentTypes"`
	EndpointURL           *string               `json:"endpointUrl"`
	DefaultParameters     json.RawMessage       `json:"defaultParameters"`
}

type updateParserConfigRequest struct {
	Name                  *string                `json:"name"`
	Backend               *service.ParserBackend `json:"backend"`
	Enabled               *bool                  `json:"enabled"`
	IsDefault             *bool                  `json:"isDefault"`
	Concurrency           *int                   `json:"concurrency"`
	SupportedContentTypes *[]string              `json:"supportedContentTypes"`
	EndpointURL           json.RawMessage        `json:"endpointUrl"`
	DefaultParameters     *json.RawMessage       `json:"defaultParameters"`
}

type parserConfigResponse struct {
	ID                    string                `json:"id"`
	Name                  string                `json:"name"`
	Backend               service.ParserBackend `json:"backend"`
	Enabled               bool                  `json:"enabled"`
	IsDefault             bool                  `json:"isDefault"`
	Concurrency           int                   `json:"concurrency"`
	SupportedContentTypes []string              `json:"supportedContentTypes,omitempty"`
	EndpointURL           *string               `json:"endpointUrl"`
	DefaultParameters     json.RawMessage       `json:"defaultParameters,omitempty"`
	CreatedAt             time.Time             `json:"createdAt"`
	UpdatedAt             time.Time             `json:"updatedAt"`
}

func parserConfigFromDomain(c service.ParserConfig) parserConfigResponse {
	return parserConfigResponse{
		ID: c.ID, Name: c.Name, Backend: c.Backend, Enabled: c.Enabled, IsDefault: c.IsDefault,
		Concurrency: c.Concurrency, SupportedContentTypes: c.SupportedContentTypes,
		EndpointURL: c.EndpointURL, DefaultParameters: c.DefaultParameters,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func parserConfigsFromDomain(items []service.ParserConfig) []parserConfigResponse {
	out := make([]parserConfigResponse, len(items))
	for i, item := range items {
		out[i] = parserConfigFromDomain(item)
	}
	return out
}

func parseNullableString(raw json.RawMessage) (*string, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, true, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, true, err
	}
	return &value, true, nil
}

func (s *Server) requireParserConfigService(w http.ResponseWriter, r *http.Request) (*service.Service, bool) {
	if s.parserConfigs != nil {
		return s.parserConfigs, true
	}
	writeAppError(w, r, service.DependencyError("parser config storage is not configured; set DATABASE_URL", nil))
	return nil, false
}

func (s *Server) handleListParserConfigs(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	parserSvc, ok := s.requireParserConfigService(w, r)
	if !ok {
		return
	}
	var enabled *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"enabled": "must be a boolean"}))
			return
		}
		enabled = &value
	}
	result, err := parserSvc.ListParserConfigs(r.Context(), reqCtx, enabled)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, parserConfigsFromDomain(result.Items), reqCtx.RequestID)
}

func (s *Server) handleCreateParserConfig(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	parserSvc, ok := s.requireParserConfigService(w, r)
	if !ok {
		return
	}
	var body createParserConfigRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	result, err := parserSvc.CreateParserConfig(r.Context(), reqCtx, service.CreateParserConfigInput{
		Name: body.Name, Backend: body.Backend, Enabled: body.Enabled, IsDefault: body.IsDefault,
		Concurrency: body.Concurrency, SupportedContentTypes: body.SupportedContentTypes,
		EndpointURL: body.EndpointURL, DefaultParameters: body.DefaultParameters,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, parserConfigFromDomain(result), reqCtx.RequestID)
}

func (s *Server) handleGetParserConfig(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	parserSvc, ok := s.requireParserConfigService(w, r)
	if !ok {
		return
	}
	result, err := parserSvc.GetParserConfig(r.Context(), reqCtx, r.PathValue("parserConfigId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, parserConfigFromDomain(result), reqCtx.RequestID)
}

func (s *Server) handleUpdateParserConfig(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	parserSvc, ok := s.requireParserConfigService(w, r)
	if !ok {
		return
	}
	var body updateParserConfigRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	endpointURL, okField, err := parseNullableString(body.EndpointURL)
	if err != nil {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"endpointUrl": "must be a string or null"}))
		return
	}
	var endpointURLPatch **string
	if okField {
		endpointURLPatch = &endpointURL
	}
	result, err := parserSvc.UpdateParserConfig(r.Context(), reqCtx, service.UpdateParserConfigInput{
		ID: r.PathValue("parserConfigId"), Name: body.Name, Backend: body.Backend, Enabled: body.Enabled,
		IsDefault: body.IsDefault, Concurrency: body.Concurrency, SupportedContentTypes: body.SupportedContentTypes,
		EndpointURL: endpointURLPatch, DefaultParameters: body.DefaultParameters,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, parserConfigFromDomain(result), reqCtx.RequestID)
}

func (s *Server) handleDeleteParserConfig(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	parserSvc, ok := s.requireParserConfigService(w, r)
	if !ok {
		return
	}
	if err := parserSvc.DeleteParserConfig(r.Context(), reqCtx, r.PathValue("parserConfigId")); err != nil {
		writeAppError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
