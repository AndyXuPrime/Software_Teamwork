package attachmentclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type FileHTTPConfig struct {
	BaseURL      string
	ServiceToken string
	Timeout      time.Duration
}

type FileHTTPClient struct {
	baseURL      *url.URL
	serviceToken string
	httpClient   *http.Client
}

func NewFileHTTPClient(cfg FileHTTPConfig) (*FileHTTPClient, error) {
	baseURL, err := parseBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: cfg.Timeout}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &FileHTTPClient{baseURL: baseURL, serviceToken: strings.TrimSpace(cfg.ServiceToken), httpClient: client}, nil
}

func (c *FileHTTPClient) Upload(ctx context.Context, name, contentType string, size int64, body io.Reader) (string, error) {
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     "file",
		"filename": name,
	}))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, body); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/internal/v1/files"), &payload)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	applyInternalHeaders(req, c.serviceToken, "", service.RequestIDFromContext(ctx))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		io.Copy(io.Discard, res.Body)
		return "", fmt.Errorf("file service returned %d", res.StatusCode)
	}
	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&envelope); err != nil {
		return "", err
	}
	if strings.TrimSpace(envelope.Data.ID) == "" {
		return "", errors.New("file service returned empty id")
	}
	return envelope.Data.ID, nil
}

func (c *FileHTTPClient) Read(ctx context.Context, fileRef string) ([]byte, error) {
	fileRef = strings.TrimSpace(fileRef)
	if fileRef == "" {
		return nil, errors.New("file reference is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("/internal/v1/files/"+url.PathEscape(fileRef)+"/content"), nil)
	if err != nil {
		return nil, err
	}
	applyInternalHeaders(req, c.serviceToken, "", service.RequestIDFromContext(ctx))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		io.Copy(io.Discard, res.Body)
		return nil, fmt.Errorf("file content read returned %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 21<<20))
}

func (c *FileHTTPClient) Delete(ctx context.Context, fileRef string) error {
	if strings.TrimSpace(fileRef) == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint("/internal/v1/files/"+url.PathEscape(fileRef)), nil)
	if err != nil {
		return err
	}
	applyInternalHeaders(req, c.serviceToken, "", service.RequestIDFromContext(ctx))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)
	if res.StatusCode == http.StatusNoContent || res.StatusCode == http.StatusNotFound {
		return nil
	}
	return fmt.Errorf("file delete returned %d", res.StatusCode)
}

func (c *FileHTTPClient) endpoint(suffix string) string {
	clone := *c.baseURL
	clone.Path = path.Join(c.baseURL.Path, suffix)
	return clone.String()
}

type ParserHTTPConfig struct {
	BaseURL      string
	ServiceToken string
	Timeout      time.Duration
}

type ParserHTTPClient struct {
	baseURL      *url.URL
	serviceToken string
	httpClient   *http.Client
}

func NewParserHTTPClient(cfg ParserHTTPConfig) (*ParserHTTPClient, error) {
	baseURL, err := parseBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	client := &http.Client{Timeout: cfg.Timeout}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &ParserHTTPClient{baseURL: baseURL, serviceToken: strings.TrimSpace(cfg.ServiceToken), httpClient: client}, nil
}

func (c *ParserHTTPClient) Parse(ctx context.Context, filename, contentType string, data []byte) (service.ParsedAttachment, error) {
	payload := map[string]any{
		"documentName": filename,
		"contentType":  contentType,
		"sizeBytes":    len(data),
		"dataBase64":   base64.StdEncoding.EncodeToString(data),
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/internal/v1/parsed-documents"), bytes.NewReader(body))
	if err != nil {
		return service.ParsedAttachment{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	applyInternalHeaders(req, c.serviceToken, "", service.RequestIDFromContext(ctx))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return service.ParsedAttachment{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		io.Copy(io.Discard, res.Body)
		return service.ParsedAttachment{}, fmt.Errorf("parser returned %d", res.StatusCode)
	}
	var envelope struct {
		Data struct {
			Pages []struct {
				PageNumber int    `json:"pageNumber"`
				Content    string `json:"content"`
			} `json:"pages"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(&envelope); err != nil {
		return service.ParsedAttachment{}, err
	}
	chunks := make([]service.ParsedAttachmentChunk, 0, len(envelope.Data.Pages))
	for _, page := range envelope.Data.Pages {
		content := strings.TrimSpace(page.Content)
		if content == "" {
			continue
		}
		chunks = append(chunks, service.ParsedAttachmentChunk{
			PageNumber: page.PageNumber,
			Content:    content,
		})
	}
	if len(chunks) == 0 {
		return service.ParsedAttachment{}, fmt.Errorf("parser returned no content")
	}
	pageCount := len(envelope.Data.Pages)
	if pageCount == 0 {
		pageCount = 1
	}
	return service.ParsedAttachment{PageCount: pageCount, Chunks: chunks}, nil
}

func (c *ParserHTTPClient) endpoint(suffix string) string {
	clone := *c.baseURL
	clone.Path = path.Join(c.baseURL.Path, suffix)
	return clone.String()
}

func parseBaseURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("base URL must be an absolute http(s) URL")
	}
	return parsed, nil
}

func applyInternalHeaders(req *http.Request, token, userID, requestID string) {
	req.Header.Set("X-Caller-Service", "qa")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("X-Service-Token", strings.TrimSpace(token))
	}
	if strings.TrimSpace(userID) != "" {
		req.Header.Set("X-User-Id", strings.TrimSpace(userID))
	}
	if strings.TrimSpace(requestID) != "" {
		req.Header.Set("X-Request-Id", strings.TrimSpace(requestID))
	}
}
