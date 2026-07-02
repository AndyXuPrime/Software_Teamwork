package mcpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	DefaultPath        = "/mcp"
	defaultTokenHeader = "Authorization"
	serverName         = "document-mcp"
	serverVersion      = "0.1.0"
)

type ToolService interface {
	ListTools(context.Context) []service.MCPToolDefinition
	CallTool(context.Context, service.RequestContext, string, json.RawMessage) service.MCPToolCallResult
}

type Config struct {
	ToolService  ToolService
	ServiceToken string
	TokenHeader  string
	Logger       *slog.Logger
}

func NewHandler(cfg Config) http.Handler {
	tokenHeader := strings.TrimSpace(cfg.TokenHeader)
	if tokenHeader == "" {
		tokenHeader = defaultTokenHeader
	}
	stream := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		if cfg.ToolService == nil {
			return nil
		}
		return newMCPServer(r.Context(), cfg.ToolService, requestContextFromRequest(r, tokenHeader))
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		Logger:       cfg.Logger,
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := strings.TrimSpace(cfg.ServiceToken); token != "" {
			if got := tokenFromHeader(r.Header, tokenHeader); got != token {
				http.Error(w, "unauthorized MCP request", http.StatusUnauthorized)
				return
			}
		}
		stream.ServeHTTP(w, r)
	})
}

func newMCPServer(ctx context.Context, toolService ToolService, reqCtx service.RequestContext) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)
	for _, definition := range toolService.ListTools(ctx) {
		definition := definition
		server.AddTool(&mcp.Tool{
			Name:        definition.Name,
			Description: definition.Description,
			InputSchema: definition.InputSchema,
		}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments
			if len(args) == 0 {
				args = json.RawMessage(`{}`)
			}
			result := toolService.CallTool(ctx, reqCtx, req.Params.Name, args)
			text, err := json.Marshal(result)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content:           []mcp.Content{&mcp.TextContent{Text: string(text)}},
				StructuredContent: result,
				IsError:           result.Error != nil || strings.EqualFold(result.Status, "failed"),
			}, nil
		})
	}
	return server
}

func requestContextFromRequest(r *http.Request, tokenHeader string) service.RequestContext {
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = newRequestID()
	}
	serviceToken := tokenFromHeader(r.Header, tokenHeader)
	if serviceToken == "" {
		serviceToken = strings.TrimSpace(r.Header.Get("X-Service-Token"))
	}
	return service.RequestContext{
		RequestID:      requestID,
		UserID:         strings.TrimSpace(r.Header.Get("X-User-Id")),
		CallerService:  strings.TrimSpace(r.Header.Get("X-Caller-Service")),
		ServiceToken:   serviceToken,
		Roles:          splitCSV(r.Header.Get("X-User-Roles")),
		Permissions:    splitCSV(r.Header.Get("X-User-Permissions")),
		ForwardedFor:   strings.TrimSpace(r.Header.Get("X-Forwarded-For")),
		ForwardedProto: strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")),
	}
}

func tokenFromHeader(header http.Header, tokenHeader string) string {
	value := strings.TrimSpace(header.Get(tokenHeader))
	if strings.EqualFold(tokenHeader, "Authorization") {
		parts := strings.Fields(value)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
		return value
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func newRequestID() string {
	data := make([]byte, 8)
	if _, err := rand.Read(data); err != nil {
		return "req_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return "req_" + hex.EncodeToString(data)
}
