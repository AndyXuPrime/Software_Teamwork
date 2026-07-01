package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapter"
)

// Bridge invokes adapter HTTP handlers in-process without loopback networking.
type Bridge struct {
	handler http.Handler
}

func NewBridge(server *adapter.Server) *Bridge {
	return &Bridge{handler: server.Handler()}
}

type adapterSuccessEnvelope struct {
	Data      json.RawMessage `json:"data"`
	Page      json.RawMessage `json:"page,omitempty"`
	RequestID string          `json:"requestId"`
}

type adapterErrorEnvelope struct {
	Error struct {
		Code      string            `json:"code"`
		Message   string            `json:"message"`
		RequestID string            `json:"requestId"`
		Fields    map[string]string `json:"fields,omitempty"`
	} `json:"error"`
}

type adapterListResult struct {
	Data json.RawMessage
	Page json.RawMessage
}

type MultipartFile struct {
	FieldName   string
	FileName    string
	Content     []byte
	ContentType string
}

func (b *Bridge) Do(ctx context.Context, caller CallerContext, method, path string, body []byte) (int, []byte, http.Header, error) {
	if b == nil || b.handler == nil {
		return http.StatusInternalServerError, nil, nil, fmt.Errorf("adapter bridge is not configured")
	}
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req = req.WithContext(ctx)
	caller.applyHeaders(req)
	if len(body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	b.handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes(), rec.Header(), nil
}

func (b *Bridge) DoGET(ctx context.Context, caller CallerContext, path string, query url.Values) (int, []byte, http.Header, error) {
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}
	return b.Do(ctx, caller, http.MethodGet, path, nil)
}

func (b *Bridge) DoJSON(ctx context.Context, caller CallerContext, method, path string, payload any) (int, []byte, http.Header, error) {
	var body []byte
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, nil, fmt.Errorf("encode request body: %w", err)
		}
		body = raw
	}
	return b.Do(ctx, caller, method, path, body)
}

func (b *Bridge) DoMultipart(ctx context.Context, caller CallerContext, method, path string, fields map[string]string, files []MultipartFile) (int, []byte, http.Header, error) {
	if b == nil || b.handler == nil {
		return http.StatusInternalServerError, nil, nil, fmt.Errorf("adapter bridge is not configured")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return 0, nil, nil, fmt.Errorf("write multipart field: %w", err)
		}
	}
	for _, file := range files {
		fieldName := file.FieldName
		if fieldName == "" {
			fieldName = "file"
		}
		part, err := writer.CreateFormFile(fieldName, file.FileName)
		if err != nil {
			return 0, nil, nil, fmt.Errorf("create multipart file: %w", err)
		}
		if _, err := part.Write(file.Content); err != nil {
			return 0, nil, nil, fmt.Errorf("write multipart file: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return 0, nil, nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req := httptest.NewRequest(method, path, body)
	req = req.WithContext(ctx)
	caller.applyHeaders(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	b.handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes(), rec.Header(), nil
}

func decodeAdapterSuccess(body []byte) (json.RawMessage, error) {
	data, _, err := decodeAdapterEnvelope(body)
	return data, err
}

func decodeAdapterList(body []byte) (adapterListResult, error) {
	data, page, err := decodeAdapterEnvelope(body)
	if err != nil {
		return adapterListResult{}, err
	}
	return adapterListResult{Data: data, Page: page}, nil
}

func decodeAdapterEnvelope(body []byte) (json.RawMessage, json.RawMessage, error) {
	var envelope adapterSuccessEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, nil, fmt.Errorf("decode adapter response: %w", err)
	}
	return envelope.Data, envelope.Page, nil
}

func adapterErrorMessage(status int, body []byte) error {
	var envelope adapterErrorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("adapter request failed with status %d", status)
	}
	msg := strings.TrimSpace(envelope.Error.Message)
	if msg == "" {
		msg = "adapter request failed"
	}
	if len(envelope.Error.Fields) > 0 {
		return fmt.Errorf("%s: %v", msg, envelope.Error.Fields)
	}
	return fmt.Errorf("%s", msg)
}

func rawToMap(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode adapter data: %w", err)
	}
	return out, nil
}

func rawToSlice(raw json.RawMessage) ([]map[string]any, error) {
	if len(raw) == 0 {
		return []map[string]any{}, nil
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode adapter list data: %w", err)
	}
	return out, nil
}
