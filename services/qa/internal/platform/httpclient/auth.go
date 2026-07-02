package httpclient

import (
	"net/http"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
)

// HeaderTransport injects one configured credential header without mutating
// the caller's request. Empty tokens are allowed for local development.
type HeaderTransport struct {
	Base          http.RoundTripper
	Header        string
	Token         string
	CallerService string
}

func (t HeaderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	if t.Token != "" {
		value := t.Token
		if strings.EqualFold(t.Header, "Authorization") && !strings.HasPrefix(strings.ToLower(value), "bearer ") {
			value = "Bearer " + value
		}
		cloned.Header.Set(t.Header, value)
	}
	setIfPresent(cloned.Header, "X-Caller-Service", t.CallerService)
	setIfPresent(cloned.Header, "X-Request-Id", contextutil.RequestIDFromContext(request.Context()))
	setIfPresent(cloned.Header, "X-User-Id", contextutil.UserIDFromContext(request.Context()))
	setIfPresent(cloned.Header, "X-User-Roles", contextutil.UserRolesFromContext(request.Context()))
	setIfPresent(cloned.Header, "X-User-Permissions", contextutil.UserPermissionsFromContext(request.Context()))
	return base.RoundTrip(cloned)
}

func setIfPresent(header http.Header, name, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	header.Set(name, value)
}
