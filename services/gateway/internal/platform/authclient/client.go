package authclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/service"
)

type Client struct {
	baseURL                  *url.URL
	serviceToken             string
	gatewayAdminServiceToken string
	httpClient               *http.Client
}

type ForwardingContext struct {
	ForwardedFor   string
	ForwardedProto string
	UserID         string
	Roles          []string
	Permissions    []string
}

type ErrorDetail struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"requestId"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type RemoteError struct {
	Status int
	Detail ErrorDetail
}

func (e *RemoteError) Error() string {
	if e == nil {
		return ""
	}
	if e.Detail.Message != "" {
		return e.Detail.Message
	}
	return fmt.Sprintf("auth service returned HTTP %d", e.Status)
}

func New(baseURL string, serviceToken string, gatewayAdminServiceToken string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, nil
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse auth base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("auth base URL must include scheme and host")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL:                  parsed,
		serviceToken:             strings.TrimSpace(serviceToken),
		gatewayAdminServiceToken: strings.TrimSpace(gatewayAdminServiceToken),
		httpClient:               &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) CreateUser(ctx context.Context, requestID string, body []byte, forwarding ForwardingContext) (service.SessionResponse, error) {
	var envelope service.SessionEnvelope
	if err := c.doJSON(ctx, http.MethodPost, "/internal/v1/users", requestID, body, http.StatusCreated, forwarding, &envelope); err != nil {
		return service.SessionResponse{}, err
	}
	return envelope.Data, nil
}

func (c *Client) CreateSession(ctx context.Context, requestID string, body []byte, forwarding ForwardingContext) (service.SessionResponse, error) {
	var envelope service.SessionEnvelope
	if err := c.doJSON(ctx, http.MethodPost, "/internal/v1/sessions", requestID, body, http.StatusOK, forwarding, &envelope); err != nil {
		return service.SessionResponse{}, err
	}
	return envelope.Data, nil
}

func (c *Client) GetUser(ctx context.Context, requestID string, userID string, forwarding ForwardingContext) (service.UserRecord, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return service.UserRecord{}, &RemoteError{Status: http.StatusUnauthorized, Detail: ErrorDetail{Code: "unauthorized", Message: "invalid user"}}
	}
	var envelope service.UserRecordEnvelope
	if err := c.doJSON(ctx, http.MethodGet, "/internal/v1/users/"+url.PathEscape(userID), requestID, nil, http.StatusOK, forwarding, &envelope); err != nil {
		return service.UserRecord{}, err
	}
	return envelope.Data, nil
}

func (c *Client) GetSession(ctx context.Context, requestID string, sessionID string, forwarding ForwardingContext) (service.SessionIdentity, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return service.SessionIdentity{}, &RemoteError{Status: http.StatusUnauthorized, Detail: ErrorDetail{Code: "unauthorized", Message: "invalid session"}}
	}
	var envelope service.SessionIdentityEnvelope
	if err := c.doJSON(ctx, http.MethodGet, "/internal/v1/sessions/"+url.PathEscape(sessionID), requestID, nil, http.StatusOK, forwarding, &envelope); err != nil {
		return service.SessionIdentity{}, err
	}
	return envelope.Data, nil
}

func (c *Client) DeleteSession(ctx context.Context, requestID string, sessionID string, forwarding ForwardingContext) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return &RemoteError{Status: http.StatusUnauthorized, Detail: ErrorDetail{Code: "unauthorized", Message: "invalid session"}}
	}
	return c.doJSON(ctx, http.MethodDelete, "/internal/v1/sessions/"+url.PathEscape(sessionID), requestID, nil, http.StatusNoContent, forwarding, nil)
}

func (c *Client) UpdateUserProfile(ctx context.Context, requestID string, userID string, body []byte, forwarding ForwardingContext) (service.UserRecord, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return service.UserRecord{}, &RemoteError{Status: http.StatusUnauthorized, Detail: ErrorDetail{Code: "unauthorized", Message: "invalid user"}}
	}
	var envelope service.UserRecordEnvelope
	if err := c.doGatewayJSON(ctx, http.MethodPatch, "/internal/v1/users/"+url.PathEscape(userID)+"/profile", requestID, body, http.StatusOK, forwarding, &envelope); err != nil {
		return service.UserRecord{}, err
	}
	return envelope.Data, nil
}

func (c *Client) ChangeUserPassword(ctx context.Context, requestID string, userID string, body []byte, forwarding ForwardingContext) (service.UserRecord, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return service.UserRecord{}, &RemoteError{Status: http.StatusUnauthorized, Detail: ErrorDetail{Code: "unauthorized", Message: "invalid user"}}
	}
	var envelope service.UserRecordEnvelope
	if err := c.doGatewayJSON(ctx, http.MethodPost, "/internal/v1/users/"+url.PathEscape(userID)+"/password-changes", requestID, body, http.StatusOK, forwarding, &envelope); err != nil {
		return service.UserRecord{}, err
	}
	return envelope.Data, nil
}

func (c *Client) CheckReady(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodGet, "/readyz", "", nil, http.StatusOK, ForwardingContext{}, nil)
}

func (c *Client) doJSON(ctx context.Context, method string, path string, requestID string, body []byte, successStatus int, forwarding ForwardingContext, target any) error {
	return c.doJSONWithToken(ctx, method, path, requestID, body, successStatus, forwarding, target, c.serviceToken)
}

func (c *Client) doGatewayJSON(ctx context.Context, method string, path string, requestID string, body []byte, successStatus int, forwarding ForwardingContext, target any) error {
	return c.doJSONWithToken(ctx, method, path, requestID, body, successStatus, forwarding, target, c.gatewayAdminServiceToken)
}

func (c *Client) doJSONWithToken(ctx context.Context, method string, path string, requestID string, body []byte, successStatus int, forwarding ForwardingContext, target any, serviceToken string) error {
	if c == nil || c.baseURL == nil {
		return fmt.Errorf("auth client is not configured")
	}

	targetURL := *c.baseURL
	targetURL.Path = joinURLPath(c.baseURL.Path, path)

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), reader)
	if err != nil {
		return err
	}
	req.Header.Set("X-Request-Id", requestID)
	req.Header.Set("X-Caller-Service", "gateway")
	if serviceToken = strings.TrimSpace(serviceToken); serviceToken != "" {
		req.Header.Set("X-Service-Token", serviceToken)
	}
	if forwardedFor := strings.TrimSpace(forwarding.ForwardedFor); forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	if forwardedProto := strings.TrimSpace(forwarding.ForwardedProto); forwardedProto != "" {
		req.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	if userID := strings.TrimSpace(forwarding.UserID); userID != "" {
		req.Header.Set("X-User-Id", userID)
	}
	if len(forwarding.Roles) > 0 {
		req.Header.Set("X-User-Roles", strings.Join(cleanStrings(forwarding.Roles), ","))
	}
	if len(forwarding.Permissions) > 0 {
		req.Header.Set("X-User-Permissions", strings.Join(cleanStrings(forwarding.Permissions), ","))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != successStatus {
		return decodeRemoteError(res)
	}
	if target == nil {
		io.Copy(io.Discard, res.Body)
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}
	return nil
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func decodeRemoteError(res *http.Response) error {
	var envelope struct {
		Error ErrorDetail `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		envelope.Error = ErrorDetail{
			Code:    "dependency_error",
			Message: "auth service returned an invalid error response",
		}
	}
	return &RemoteError{Status: res.StatusCode, Detail: envelope.Error}
}

func joinURLPath(base string, path string) string {
	base = strings.TrimRight(base, "/")
	path = "/" + strings.TrimLeft(path, "/")
	if base == "" {
		return path
	}
	return base + path
}
