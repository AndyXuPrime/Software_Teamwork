package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAppVersionFreshnessReturnsCurrentWhenBuildIncludesDevelop(t *testing.T) {
	const latestSHA = "abcdef1234567890"
	const currentSHA = "fedcba0987654321"
	var capturedAuthorization string
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthorization = r.Header.Get("Authorization")
		if r.URL.Path != "/compare/develop..."+currentSHA {
			t.Fatalf("github path = %q", r.URL.Path)
		}
		writeAppVersionCompareResponse(t, w, latestSHA, 1, 0)
	}))
	defer github.Close()

	server := newAppVersionTestServer(t, github.URL, "backend-github-token")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha="+currentSHA, nil)
	req.Header.Set("X-Request-Id", "req_app_version")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if capturedAuthorization != "Bearer backend-github-token" {
		t.Fatalf("GitHub Authorization header = %q", capturedAuthorization)
	}
	if strings.Contains(res.Body.String(), "backend-github-token") {
		t.Fatalf("backend token leaked into response: %s", res.Body.String())
	}
	var body appVersionEnvelope
	decodeAppVersionJSON(t, res.Body, &body)
	if body.RequestID != "req_app_version" {
		t.Fatalf("requestId = %q", body.RequestID)
	}
	if body.Data.Status != appFreshnessCurrent ||
		body.Data.CurrentSHA != currentSHA ||
		body.Data.LatestSHA != latestSHA ||
		body.Data.LatestURL == "" {
		t.Fatalf("freshness = %+v", body.Data)
	}
}

func TestAppVersionFreshnessReturnsDifferentWhenBuildIsBehindDevelop(t *testing.T) {
	const latestSHA = "abcdef1234567890"
	const currentSHA = "1111111111111111"
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compare/develop..."+currentSHA {
			t.Fatalf("github path = %q", r.URL.Path)
		}
		writeAppVersionCompareResponse(t, w, latestSHA, 0, 2)
	}))
	defer github.Close()

	server := newAppVersionTestServer(t, github.URL, "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha="+currentSHA, nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body appVersionEnvelope
	decodeAppVersionJSON(t, res.Body, &body)
	if body.Data.Status != appFreshnessDifferent ||
		body.Data.CurrentSHA != currentSHA ||
		body.Data.LatestSHA != latestSHA {
		t.Fatalf("freshness = %+v", body.Data)
	}
}

func TestAppVersionFreshnessFallsBackToUnknownOnGitHubForbidden(t *testing.T) {
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
	}))
	defer github.Close()

	server := newAppVersionTestServer(t, github.URL, "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha=abcdef", nil)
	req.Header.Set("X-Request-Id", "req_app_version_403")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body appVersionEnvelope
	decodeAppVersionJSON(t, res.Body, &body)
	if body.Data.Status != appFreshnessUnknown ||
		body.Data.Reason != appFreshnessReasonGitHub403 ||
		body.Data.CurrentSHA != "abcdef" ||
		body.Data.LatestSHA != "" {
		t.Fatalf("freshness = %+v", body.Data)
	}
}

func TestAppVersionFreshnessCachesFreshnessByCurrentSHA(t *testing.T) {
	const latestSHA = "abcdef1234567890"
	const currentSHA = "1111111111111111"
	calls := 0
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeAppVersionCompareResponse(t, w, latestSHA, 0, 1)
	}))
	defer github.Close()

	server := newAppVersionTestServer(t, github.URL, "")
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha="+currentSHA, nil)
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("GitHub calls = %d, want 1", calls)
	}
}

func TestAppVersionFreshnessCoalescesConcurrentGitHubRequests(t *testing.T) {
	const latestSHA = "abcdef1234567890"
	const currentSHA = "1111111111111111"
	var calls int64
	started := make(chan struct{})
	release := make(chan struct{})
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compare/develop..."+currentSHA {
			t.Fatalf("github path = %q", r.URL.Path)
		}
		if atomic.AddInt64(&calls, 1) == 1 {
			close(started)
		}
		<-release
		writeAppVersionCompareResponse(t, w, latestSHA, 0, 1)
	}))
	defer github.Close()

	checker := newAppVersionTestChecker(t, github.URL, "")
	var wg sync.WaitGroup
	begin := make(chan struct{})
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-begin
			freshness := checker.CheckFreshness(context.Background(), currentSHA)
			if freshness.Status != appFreshnessDifferent {
				t.Errorf("status = %q, want %q", freshness.Status, appFreshnessDifferent)
			}
		}()
	}

	close(begin)
	<-started
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if calls != 1 {
		t.Fatalf("GitHub calls = %d, want 1", calls)
	}
}

func TestAppVersionFreshnessRejectsLongCurrentSHA(t *testing.T) {
	server := newAppVersionTestServer(t, "https://github.test/not-called", "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha="+strings.Repeat("a", 129), nil)
	req.Header.Set("X-Request-Id", "req_app_version_validation")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	decodeAppVersionJSON(t, res.Body, &body)
	if body.Error.Code != "validation_error" || body.Error.RequestID != "req_app_version_validation" {
		t.Fatalf("error = %+v", body.Error)
	}
}

func newAppVersionTestServer(t *testing.T, githubURL string, githubToken string) http.Handler {
	t.Helper()
	checker := newAppVersionTestChecker(t, githubURL, githubToken)
	return NewServer(Config{
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceVersion:     "test",
		Environment:        "test",
		RequestTimeout:     time.Second,
		MaxBodyBytes:       1024,
		CORSAllowedOrigins: []string{"*"},
		AppVersionChecker:  checker,
	})
}

func newAppVersionTestChecker(t *testing.T, githubURL string, githubToken string) *gitHubAppVersionChecker {
	t.Helper()
	checker := newGitHubAppVersionChecker(http.DefaultClient, slog.New(slog.NewTextHandler(io.Discard, nil)), githubToken)
	checker.apiURL = strings.TrimRight(githubURL, "/") + "/compare/develop..."
	checker.now = func() time.Time {
		return time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	}
	return checker
}

type appVersionEnvelope struct {
	Data      AppVersionFreshness `json:"data"`
	RequestID string              `json:"requestId"`
}

func decodeAppVersionJSON(t *testing.T, r io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(r).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

func writeAppVersionCompareResponse(t *testing.T, w http.ResponseWriter, latestSHA string, aheadBy int, behindBy int) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	body := map[string]any{
		"ahead_by":  aheadBy,
		"behind_by": behindBy,
		"base_commit": map[string]string{
			"sha":      latestSHA,
			"html_url": "https://github.test/commit/" + latestSHA,
		},
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}
