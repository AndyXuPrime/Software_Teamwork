package httpclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
)

func TestHeaderTransportInjectsTrustedContextHeadersWithoutToken(t *testing.T) {
	captured := make(http.Header)
	transport := HeaderTransport{
		Base: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Header.Clone()
			return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody, Header: http.Header{}}, nil
		}),
		CallerService: "qa",
	}
	ctx := context.Background()
	ctx = contextutil.WithRequestID(ctx, "req-1")
	ctx = contextutil.WithUserID(ctx, "user-1")
	ctx = contextutil.WithUserRoles(ctx, "admin")
	ctx = contextutil.WithUserPermissions(ctx, "report:write")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://document.test/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}

	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()

	for name, want := range map[string]string{
		"X-Caller-Service":   "qa",
		"X-Request-Id":       "req-1",
		"X-User-Id":          "user-1",
		"X-User-Roles":       "admin",
		"X-User-Permissions": "report:write",
	} {
		if got := captured.Get(name); got != want {
			t.Fatalf("%s=%q, want %q", name, got, want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
