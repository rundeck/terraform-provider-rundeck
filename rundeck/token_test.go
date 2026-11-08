package rundeck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRundeck is a minimal stand-in for the Rundeck endpoints _getToken uses:
// form login, and listing, deleting and creating API tokens. Like Rundeck, it
// never includes a token's value when listing tokens, and only answers API
// calls that carry the session cookie set by a successful login.
type fakeRundeck struct {
	password      string
	listStatus    int // status for GET /tokens/{user}; 0 means 200
	createNoValue bool
	dropCreate    bool

	mu       sync.Mutex
	tokens   []TokenResp
	deleted  []string
	created  []map[string]interface{}
	apiCalls int
}

const fakeSession = "fake-session"

func (f *fakeRundeck) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/j_security_check", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("j_password") != f.password {
			http.Redirect(w, r, "/user/error", http.StatusFound)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: fakeSession, Path: "/"})
		http.Redirect(w, r, "/menu/home", http.StatusFound)
	})
	mux.HandleFunc("/menu/home", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>home</html>") })
	mux.HandleFunc("/user/error", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>login failed</html>") })

	mux.HandleFunc("/api/43/tokens/", func(w http.ResponseWriter, r *http.Request) {
		if !f.authorized(w, r) {
			return
		}
		if f.listStatus != 0 {
			w.WriteHeader(f.listStatus)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		listed := make([]TokenResp, len(f.tokens))
		for i, t := range f.tokens {
			t.Token = "" // Rundeck masks token values when listing
			listed[i] = t
		}
		_ = json.NewEncoder(w).Encode(listed)
	})
	mux.HandleFunc("/api/43/token/", func(w http.ResponseWriter, r *http.Request) {
		if !f.authorized(w, r) || r.Method != http.MethodDelete {
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.deleted = append(f.deleted, strings.TrimPrefix(r.URL.Path, "/api/43/token/"))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/43/tokens", func(w http.ResponseWriter, r *http.Request) {
		if !f.authorized(w, r) || r.Method != http.MethodPost {
			return
		}
		if f.dropCreate {
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.created = append(f.created, body)
		n := len(f.created)
		f.mu.Unlock()
		resp := TokenResp{ID: fmt.Sprintf("new-%d", n), Name: terraformTokenName, User: "admin"}
		if !f.createNoValue {
			resp.Token = fmt.Sprintf("secret-value-%d", n)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})
	return mux
}

// authorized rejects API calls made without the login session cookie.
func (f *fakeRundeck) authorized(w http.ResponseWriter, r *http.Request) bool {
	f.mu.Lock()
	f.apiCalls++
	f.mu.Unlock()
	if c, err := r.Cookie("JSESSIONID"); err != nil || c.Value != fakeSession {
		w.WriteHeader(http.StatusForbidden)
		return false
	}
	return true
}

func newFakeRundeck(t *testing.T, f *fakeRundeck) string {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestGetTokenCreatesShortLivedToken(t *testing.T) {
	f := &fakeRundeck{password: "pw"}
	url := newFakeRundeck(t, f)

	token, err := _getToken(url, "admin", "pw", "test", DefaultAuthTokenDuration)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if token != "secret-value-1" {
		t.Errorf("expected the value returned by token creation, got %q", token)
	}
	if len(f.created) != 1 {
		t.Fatalf("expected one token to be created, got %d", len(f.created))
	}
	body := f.created[0]
	if body["duration"] != "1h" || body["name"] != terraformTokenName || body["user"] != "admin" {
		t.Errorf("unexpected token request: %v", body)
	}
}

func TestGetTokenUsesConfiguredDuration(t *testing.T) {
	f := &fakeRundeck{password: "pw"}
	url := newFakeRundeck(t, f)

	if _, err := _getToken(url, "admin", "pw", "test", "15m"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := f.created[0]["duration"]; got != "15m" {
		t.Errorf("expected duration 15m, got %v", got)
	}
}

// A token left by an earlier run cannot be reused, because Rundeck does not
// return its value when listing; a new token must be created every time.
func TestGetTokenDoesNotReuseListedToken(t *testing.T) {
	f := &fakeRundeck{password: "pw", tokens: []TokenResp{
		{ID: "old", Name: terraformTokenName, User: "admin", Token: "old-value", Expiration: time.Now().Add(time.Hour)},
	}}
	url := newFakeRundeck(t, f)

	token, err := _getToken(url, "admin", "pw", "test", DefaultAuthTokenDuration)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if token != "secret-value-1" {
		t.Errorf("expected a newly created token, got %q", token)
	}
	if len(f.deleted) != 0 {
		t.Errorf("a token that has not expired must not be deleted, deleted %v", f.deleted)
	}
}

func TestGetTokenRemovesOnlyExpiredTerraformTokens(t *testing.T) {
	f := &fakeRundeck{password: "pw", tokens: []TokenResp{
		{ID: "expired-tf", Name: terraformTokenName, User: "admin", Expired: true},
		{ID: "expired-tf-2", Name: terraformTokenName, User: "admin", Expired: true},
		{ID: "valid-tf", Name: terraformTokenName, User: "admin", Expired: false},
		{ID: "expired-other-name", Name: "ci-token", User: "admin", Expired: true},
		{ID: "expired-other-user", Name: terraformTokenName, User: "someone-else", Expired: true},
	}}
	url := newFakeRundeck(t, f)

	if _, err := _getToken(url, "admin", "pw", "test", DefaultAuthTokenDuration); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	sort.Strings(f.deleted)
	if got := strings.Join(f.deleted, ","); got != "expired-tf,expired-tf-2" {
		t.Errorf("expected only the user's expired terraform-token tokens to be deleted, deleted %q", got)
	}
}

func TestGetTokenCleanupFailureDoesNotBlockLogin(t *testing.T) {
	f := &fakeRundeck{password: "pw", listStatus: http.StatusForbidden}
	url := newFakeRundeck(t, f)

	token, err := _getToken(url, "admin", "pw", "test", DefaultAuthTokenDuration)
	if err != nil {
		t.Fatalf("a failed cleanup must not fail authentication: %s", err)
	}
	if token == "" {
		t.Error("expected a token")
	}
}

func TestGetTokenRejectedLogin(t *testing.T) {
	f := &fakeRundeck{password: "pw"}
	url := newFakeRundeck(t, f)

	_, err := _getToken(url, "admin", "wrong", "test", DefaultAuthTokenDuration)
	if err == nil || !strings.Contains(err.Error(), "rejected the login") {
		t.Fatalf("expected a rejected-login error, got %v", err)
	}
	if f.apiCalls != 0 {
		t.Errorf("expected no API calls after a rejected login, got %d", f.apiCalls)
	}
}

func TestGetTokenCreateRequestFailureReturnsError(t *testing.T) {
	f := &fakeRundeck{password: "pw", dropCreate: true}
	url := newFakeRundeck(t, f)

	if _, err := _getToken(url, "admin", "pw", "test", DefaultAuthTokenDuration); err == nil {
		t.Fatal("expected an error when the token request fails")
	}
}

func TestGetTokenMissingTokenValueIsError(t *testing.T) {
	f := &fakeRundeck{password: "pw", createNoValue: true}
	url := newFakeRundeck(t, f)

	if _, err := _getToken(url, "admin", "pw", "test", DefaultAuthTokenDuration); err == nil {
		t.Fatal("expected an error when Rundeck returns no token value")
	}
}

func TestValidAuthTokenDuration(t *testing.T) {
	for d, want := range map[string]bool{
		"1h": true, "30m": true, "90s": true, "2d": true, "1w": true, "1y": true,
		"": false, "0": false, "0h": false, "60": false, "1h30m": false, "1H": false, "-1h": false, " 1h": false,
	} {
		if got := validAuthTokenDuration(d); got != want {
			t.Errorf("validAuthTokenDuration(%q) = %v, want %v", d, got, want)
		}
	}
}

// accTokenEnv returns the connection settings for the token acceptance tests,
// skipping the test when acceptance tests are not enabled or no username and
// password are configured.
func accTokenEnv(t *testing.T) (url, apiVersion, username, password string) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests are skipped unless TF_ACC is set")
	}
	testAccPreCheck(t)
	username = os.Getenv("RUNDECK_AUTH_USERNAME")
	password = os.Getenv("RUNDECK_AUTH_PASSWORD")
	if username == "" || password == "" {
		t.Skip("RUNDECK_AUTH_USERNAME and RUNDECK_AUTH_PASSWORD are not set")
	}
	apiVersion = os.Getenv("RUNDECK_API_VERSION")
	if apiVersion == "" {
		apiVersion = "56"
	}
	return os.Getenv("RUNDECK_URL"), apiVersion, username, password
}

// assertTokenAuthenticates checks that token is accepted by the Rundeck API.
func assertTokenAuthenticates(t *testing.T, url, apiVersion, token string) {
	t.Helper()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/%s/system/info", url, apiVersion), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Rundeck-Auth-Token", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token was not accepted by the Rundeck API: status %d", resp.StatusCode)
	}
}

func TestAccToken(t *testing.T) {
	url, apiVersion, username, password := accTokenEnv(t)

	token, err := _getToken(url, username, password, "test", DefaultAuthTokenDuration)
	if err != nil {
		t.Fatalf("failed to get a token: %s", err)
	}
	assertTokenAuthenticates(t, url, apiVersion, token)
}

// Every run creates its own token, and expired terraform-token tokens from
// earlier runs are deleted.
func TestAccTokenNewTokenPerRunAndCleanup(t *testing.T) {
	url, apiVersion, username, password := accTokenEnv(t)

	first, err := _getToken(url, username, password, "test", "1s")
	if err != nil {
		t.Fatalf("failed to get the first token: %s", err)
	}
	time.Sleep(2 * time.Second) // let the first token expire

	second, err := _getToken(url, username, password, "test", DefaultAuthTokenDuration)
	if err != nil {
		t.Fatalf("failed to get the second token: %s", err)
	}
	if second == first {
		t.Fatal("expected a new token on the second run")
	}
	assertTokenAuthenticates(t, url, apiVersion, second)

	client := newHTTPClientWithUserAgent("test")
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/%s/tokens/%s", url, apiVersion, username), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Rundeck-Auth-Token", second)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var tokens []TokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		t.Fatalf("failed to list tokens: %s", err)
	}
	if left := expiredTerraformTokens(tokens, username); len(left) != 0 {
		t.Errorf("expected expired %s tokens to be deleted, %d remain", terraformTokenName, len(left))
	}
}
