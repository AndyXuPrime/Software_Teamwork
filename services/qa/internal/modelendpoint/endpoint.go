package modelendpoint

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

const (
	chatCompletionsPath = "/internal/v1/chat/completions"
	aiGatewayPort       = "8086"
)

// AIGatewayChatEndpoint is a canonical, trusted endpoint. It stores only
// enumerated URL literals, so request URLs are not built from caller-controlled
// host text after validation.
type AIGatewayChatEndpoint string

// NormalizeAIGatewayChatEndpoint validates the only model egress endpoint QA
// may call directly. Provider-specific base URLs and credentials belong in AI
// Gateway profiles, not in QA runtime settings.
func NormalizeAIGatewayChatEndpoint(raw string) (string, error) {
	endpoint, err := ParseAIGatewayChatEndpoint(raw)
	if err != nil {
		return "", err
	}
	return endpoint.String(), nil
}

func ParseAIGatewayChatEndpoint(raw string) (AIGatewayChatEndpoint, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("must be an absolute http(s) URL")
	}
	if parsed.User != nil {
		return "", errors.New("must not contain credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("must not contain query or fragment")
	}
	if strings.TrimRight(parsed.EscapedPath(), "/") != chatCompletionsPath {
		return "", errors.New("must target AI Gateway chat completions")
	}
	if !trustedInternalHost(parsed.Hostname()) {
		return "", errors.New("host is not trusted")
	}
	if port := parsed.Port(); port != "" && port != aiGatewayPort {
		return "", errors.New("port is not trusted")
	}
	endpoint, ok := canonicalEndpoint(parsed.Scheme, parsed.Hostname())
	if !ok {
		return "", errors.New("host is not trusted")
	}
	return endpoint, nil
}

func (e AIGatewayChatEndpoint) String() string {
	return string(e)
}

func canonicalEndpoint(schemeValue, hostValue string) (AIGatewayChatEndpoint, bool) {
	hostValue = strings.Trim(strings.ToLower(hostValue), "[]")
	if schemeValue == "https" {
		switch hostValue {
		case "localhost":
			return "https://localhost:8086/internal/v1/chat/completions", true
		case "ai-gateway":
			return "https://ai-gateway:8086/internal/v1/chat/completions", true
		case "127.0.0.1":
			return "https://127.0.0.1:8086/internal/v1/chat/completions", true
		case "::1":
			return "https://[::1]:8086/internal/v1/chat/completions", true
		}
		return "", false
	}
	switch hostValue {
	case "localhost":
		return "http://localhost:8086/internal/v1/chat/completions", true
	case "ai-gateway":
		return "http://ai-gateway:8086/internal/v1/chat/completions", true
	case "127.0.0.1":
		return "http://127.0.0.1:8086/internal/v1/chat/completions", true
	case "::1":
		return "http://[::1]:8086/internal/v1/chat/completions", true
	}
	return "", false
}

func trustedInternalHost(host string) bool {
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "" {
		return false
	}
	switch host {
	case "localhost", "ai-gateway":
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
