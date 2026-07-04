package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
)

const (
	paddleOCRBaseURLKey     = "paddleocr_base_url"
	paddleOCRAccessTokenKey = "paddleocr_access_token"
	paddleOCRAlgorithmKey   = "paddleocr_algorithm"
	paddleOCRDefaultModel   = "PP-StructureV3"
)

const (
	PaddleOCRBaseURLParameter     = paddleOCRBaseURLKey
	PaddleOCRAccessTokenParameter = paddleOCRAccessTokenKey
	PaddleOCRAlgorithmParameter   = paddleOCRAlgorithmKey
	PaddleOCRDefaultModel         = paddleOCRDefaultModel
)

func (s *Service) ListParserConfigs(ctx context.Context, reqCtx RequestContext, enabled *bool) (ParserConfigList, error) {
	if err := requireParserAdmin(reqCtx); err != nil {
		return ParserConfigList{}, err
	}
	items, err := s.repo.ListParserConfigs(ctx, enabled)
	if err != nil {
		return ParserConfigList{}, repositoryError(err)
	}
	return ParserConfigList{Items: items}, nil
}

func (s *Service) GetParserConfig(ctx context.Context, reqCtx RequestContext, id string) (ParserConfig, error) {
	if err := requireParserAdmin(reqCtx); err != nil {
		return ParserConfig{}, err
	}
	if strings.TrimSpace(id) == "" {
		return ParserConfig{}, ValidationError("request validation failed", map[string]string{"parserConfigId": "is required"})
	}
	config, err := s.repo.GetParserConfig(ctx, id)
	if err != nil {
		return ParserConfig{}, repositoryError(err)
	}
	return config, nil
}

func (s *Service) CreateParserConfig(ctx context.Context, reqCtx RequestContext, input CreateParserConfigInput) (ParserConfig, error) {
	if err := requireParserAdmin(reqCtx); err != nil {
		return ParserConfig{}, err
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	isDefault := false
	if input.IsDefault != nil {
		isDefault = *input.IsDefault
	}
	config := ParserConfig{
		ID: s.newID("parser_config"), Name: strings.TrimSpace(input.Name), Backend: input.Backend,
		Enabled: enabled, IsDefault: isDefault, Concurrency: input.Concurrency,
		SupportedContentTypes: normalizeContentTypes(input.SupportedContentTypes),
		EndpointURL:           normalizeEndpoint(input.EndpointURL), DefaultParameters: normalizeParameters(input.DefaultParameters),
		CreatedAt: s.now(), UpdatedAt: s.now(),
	}
	config = normalizeParserConfigForBackend(config, nil)
	if fields := validateParserConfig(config); len(fields) > 0 {
		return ParserConfig{}, ValidationError("request validation failed", fields)
	}
	audit := s.parserAudit(reqCtx, config.ID, "created", []string{"configuration"})
	created, err := s.repo.CreateParserConfig(ctx, config, audit)
	if err != nil {
		return ParserConfig{}, repositoryError(err)
	}
	return created, nil
}

func (s *Service) UpdateParserConfig(ctx context.Context, reqCtx RequestContext, input UpdateParserConfigInput) (ParserConfig, error) {
	if err := requireParserAdmin(reqCtx); err != nil {
		return ParserConfig{}, err
	}
	current, err := s.repo.GetParserConfig(ctx, input.ID)
	if err != nil {
		return ParserConfig{}, repositoryError(err)
	}
	previousParameters := current.DefaultParameters
	changed := make([]string, 0, 8)
	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)
		changed = append(changed, "name")
	}
	if input.Backend != nil {
		current.Backend = *input.Backend
		changed = append(changed, "backend")
	}
	if input.Enabled != nil {
		current.Enabled = *input.Enabled
		changed = append(changed, "enabled")
	}
	if input.IsDefault != nil {
		if current.IsDefault && !*input.IsDefault {
			return ParserConfig{}, ConflictError("replace the default parser config before clearing it", nil)
		}
		current.IsDefault = *input.IsDefault
		changed = append(changed, "default")
	}
	if input.Concurrency != nil {
		current.Concurrency = *input.Concurrency
		changed = append(changed, "concurrency")
	}
	if input.SupportedContentTypes != nil {
		current.SupportedContentTypes = normalizeContentTypes(*input.SupportedContentTypes)
		changed = append(changed, "content_types")
	}
	if input.EndpointURL != nil {
		current.EndpointURL = normalizeEndpoint(*input.EndpointURL)
		changed = append(changed, "endpoint")
	}
	if input.DefaultParameters != nil {
		current.DefaultParameters = normalizeParameters(*input.DefaultParameters)
		changed = append(changed, "parameters")
	}
	if len(changed) == 0 {
		return ParserConfig{}, ValidationError("request validation failed", map[string]string{"body": "must contain at least one field"})
	}
	current = normalizeParserConfigForBackend(current, previousParameters)
	current.UpdatedAt = s.now()
	if fields := validateParserConfig(current); len(fields) > 0 {
		return ParserConfig{}, ValidationError("request validation failed", fields)
	}
	sort.Strings(changed)
	updated, err := s.repo.UpdateParserConfig(ctx, current, s.parserAudit(reqCtx, current.ID, "updated", changed))
	if err != nil {
		return ParserConfig{}, repositoryError(err)
	}
	return updated, nil
}

func (s *Service) DeleteParserConfig(ctx context.Context, reqCtx RequestContext, id string) error {
	if err := requireParserAdmin(reqCtx); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" {
		return ValidationError("request validation failed", map[string]string{"parserConfigId": "is required"})
	}
	now := s.now()
	if err := s.repo.SoftDeleteParserConfig(ctx, id, now, s.parserAudit(reqCtx, id, "disabled", []string{"enabled"})); err != nil {
		return repositoryError(err)
	}
	return nil
}

func (s *Service) ResolveParserConfig(ctx context.Context, contentType string) (ParserConfigSnapshot, error) {
	config, err := s.repo.GetEffectiveParserConfig(ctx, strings.TrimSpace(contentType))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return defaultBuiltinParserSnapshot(), nil
		}
		return ParserConfigSnapshot{}, repositoryError(err)
	}
	if fields := validateParserConfig(config); len(fields) > 0 {
		return ParserConfigSnapshot{}, ConflictError("effective parser config is invalid", nil)
	}
	return ParserConfigSnapshot{ParserConfigID: config.ID, Backend: config.Backend, Concurrency: config.Concurrency,
		SupportedContentTypes: append([]string(nil), config.SupportedContentTypes...), EndpointURL: cloneString(config.EndpointURL),
		DefaultParameters: cloneRaw(config.DefaultParameters)}, nil
}

func (s *Service) ResolveParserConfigByID(ctx context.Context, id string) (ParserConfigSnapshot, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ParserConfigSnapshot{}, ValidationError("request validation failed", map[string]string{"parserConfigId": "is required"})
	}
	config, err := s.repo.GetParserConfig(ctx, id)
	if err != nil {
		return ParserConfigSnapshot{}, repositoryError(err)
	}
	if !config.Enabled {
		return ParserConfigSnapshot{}, ConflictError("parser config is disabled", nil)
	}
	if fields := validateParserConfig(config); len(fields) > 0 {
		return ParserConfigSnapshot{}, ConflictError("parser config is invalid", nil)
	}
	return ParserConfigSnapshot{ParserConfigID: config.ID, Backend: config.Backend, Concurrency: config.Concurrency,
		SupportedContentTypes: append([]string(nil), config.SupportedContentTypes...), EndpointURL: cloneString(config.EndpointURL),
		DefaultParameters: cloneRaw(config.DefaultParameters)}, nil
}

func defaultBuiltinParserSnapshot() ParserConfigSnapshot {
	return ParserConfigSnapshot{
		Backend:               ParserBackendBuiltin,
		Concurrency:           4,
		SupportedContentTypes: []string{"application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "text/markdown", "text/plain"},
		DefaultParameters:     json.RawMessage(`{}`),
	}
}

func marshalParserConfigSnapshot(snapshot ParserConfigSnapshot) (json.RawMessage, error) {
	body, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func requireParserAdmin(reqCtx RequestContext) error {
	if strings.TrimSpace(reqCtx.UserID) == "" {
		return UnauthorizedError()
	}
	if hasAdminRole(reqCtx.Roles) ||
		hasPermission(reqCtx.Permissions, PermissionSystemAdmin) ||
		hasPermission(reqCtx.Permissions, PermissionKnowledgeAdmin) ||
		hasPermission(reqCtx.Permissions, PermissionAdminParserConfig) {
		return nil
	}
	return ForbiddenError("knowledge administration permission is required")
}

func validateParserConfig(config ParserConfig) map[string]string {
	fields := map[string]string{}
	if config.Name == "" {
		fields["name"] = "is required"
	} else if len(config.Name) > 120 {
		fields["name"] = "must be at most 120 characters"
	}
	switch config.Backend {
	case ParserBackendBuiltin, ParserBackendTika, ParserBackendUnstructured, ParserBackendLocalOCR, ParserBackendRemoteCompatible, ParserBackendPaddleOCRCloud:
	default:
		fields["backend"] = "is not supported"
	}
	if config.Concurrency < 1 || config.Concurrency > 128 {
		fields["concurrency"] = "must be between 1 and 128"
	}
	if config.IsDefault && !config.Enabled {
		fields["isDefault"] = "default config must be enabled"
	}
	if !validParameterObject(config.DefaultParameters) {
		fields["defaultParameters"] = "must be a valid JSON object"
	}
	if config.Backend == ParserBackendRemoteCompatible {
		if config.EndpointURL == nil {
			fields["endpointUrl"] = "is required for remote_compatible backend"
		} else if !validEndpoint(*config.EndpointURL) {
			fields["endpointUrl"] = "must be an absolute http or https URI without credentials"
		}
	} else if config.EndpointURL != nil && !validEndpoint(*config.EndpointURL) {
		fields["endpointUrl"] = "must be an absolute http or https URI without credentials"
	}
	if config.Backend == ParserBackendPaddleOCRCloud {
		params := parserParameterObject(config.DefaultParameters)
		baseURL := parserParameterString(params, paddleOCRBaseURLKey)
		if baseURL == "" {
			fields[paddleOCRBaseURLKey] = "is required for paddleocr_cloud backend"
		} else if !validEndpoint(baseURL) {
			fields[paddleOCRBaseURLKey] = "must be an absolute http or https URI without credentials"
		}
		if parserParameterString(params, paddleOCRAccessTokenKey) == "" {
			fields[paddleOCRAccessTokenKey] = "is required for paddleocr_cloud backend"
		}
	}
	for _, value := range config.SupportedContentTypes {
		if !strings.Contains(value, "/") {
			fields["supportedContentTypes"] = "must contain valid media types"
			break
		}
	}
	return fields
}

func normalizeParserConfigForBackend(config ParserConfig, previousParameters json.RawMessage) ParserConfig {
	config.DefaultParameters = normalizeParameters(config.DefaultParameters)
	if config.Backend != ParserBackendRemoteCompatible {
		config.EndpointURL = nil
	}
	if config.Backend == ParserBackendPaddleOCRCloud {
		config.DefaultParameters = normalizePaddleOCRParameters(config.DefaultParameters, previousParameters)
		return config
	}
	config.DefaultParameters = removePaddleOCRParameters(config.DefaultParameters)
	return config
}

func normalizePaddleOCRParameters(raw, previousRaw json.RawMessage) json.RawMessage {
	params := parserParameterObject(raw)
	if params == nil {
		params = map[string]any{}
	}
	baseURL := parserParameterString(params, paddleOCRBaseURLKey)
	accessToken := parserParameterString(params, paddleOCRAccessTokenKey)
	if accessToken == "" {
		accessToken = parserParameterString(parserParameterObject(previousRaw), paddleOCRAccessTokenKey)
	}
	algorithm := parserParameterString(params, paddleOCRAlgorithmKey)
	if algorithm == "" {
		algorithm = paddleOCRDefaultModel
	}

	setOrDeleteString(params, paddleOCRBaseURLKey, baseURL)
	setOrDeleteString(params, paddleOCRAccessTokenKey, accessToken)
	params[paddleOCRAlgorithmKey] = algorithm
	return marshalParameterObject(params)
}

func removePaddleOCRParameters(raw json.RawMessage) json.RawMessage {
	params := parserParameterObject(raw)
	if params == nil {
		return normalizeParameters(raw)
	}
	delete(params, paddleOCRBaseURLKey)
	delete(params, paddleOCRAccessTokenKey)
	delete(params, paddleOCRAlgorithmKey)
	return marshalParameterObject(params)
}

func parserParameterObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil
	}
	return params
}

func parserParameterString(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	value, ok := params[key]
	if !ok {
		return ""
	}
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(raw)
}

func setOrDeleteString(params map[string]any, key, value string) {
	if strings.TrimSpace(value) == "" {
		delete(params, key)
		return
	}
	params[key] = strings.TrimSpace(value)
}

func marshalParameterObject(params map[string]any) json.RawMessage {
	body, err := json.Marshal(params)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return body
}

func PaddleOCRAccessTokenConfigured(raw json.RawMessage) bool {
	return parserParameterString(parserParameterObject(raw), paddleOCRAccessTokenKey) != ""
}

func RedactParserConfigDefaultParameters(raw json.RawMessage) json.RawMessage {
	params := parserParameterObject(raw)
	if params == nil {
		return normalizeParameters(raw)
	}
	sanitized, ok := sanitizeParserParameterValue(params)
	if !ok {
		return json.RawMessage(`{}`)
	}
	out, ok := sanitized.(map[string]any)
	if !ok {
		return json.RawMessage(`{}`)
	}
	return marshalParameterObject(out)
}

func IsSensitiveParserParameterKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"secret", "password", "credential", "api_key", "apikey", "access_key", "accesskey", "private_key", "privatekey", "access_token", "accesstoken", "auth_token", "authtoken", "refresh_token", "refreshtoken", "bearer_token", "bearertoken"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return normalized == "token"
}

func sanitizeParserParameterValue(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key == "" || IsSensitiveParserParameterKey(key) {
				continue
			}
			if sanitized, ok := sanitizeParserParameterValue(value); ok {
				out[key] = sanitized
			}
		}
		return out, true
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			if sanitized, ok := sanitizeParserParameterValue(item); ok {
				out = append(out, sanitized)
			}
		}
		return out, true
	default:
		return value, true
	}
}

func validParameterObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return value != nil
}

func validEndpoint(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	return err == nil && parsed.IsAbs() && parsed.Host != "" && parsed.User == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func normalizeEndpoint(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
func normalizeParameters(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return cloneRaw(value)
}
func normalizeContentTypes(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func (s *Service) parserAudit(reqCtx RequestContext, id, action string, changed []string) ParserConfigAudit {
	summary, _ := json.Marshal(map[string]any{"action": action, "changedFields": changed})
	return ParserConfigAudit{ID: s.newID("audit"), ParserConfigID: id, ActorUserID: reqCtx.UserID, Action: action, Summary: summary, CreatedAt: s.now()}
}
