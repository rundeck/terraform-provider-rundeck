package rundeck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	// terraformTokenName is the name given to the API tokens the provider
	// creates when it authenticates with auth_username/auth_password.
	terraformTokenName = "terraform-token"

	// tokenAPIVersion is the API version used for the token endpoints.
	tokenAPIVersion = "43"

	// DefaultAuthTokenDuration is the lifetime of the API token the provider
	// creates for a username/password login when auth_token_duration is unset.
	DefaultAuthTokenDuration = "1h"
)

// authTokenDurationPattern matches the durations accepted for
// auth_token_duration: a positive whole number followed by one unit, which is
// the form Rundeck's token API accepts (it rejects compound values such as
// "1h30m"). A zero or unitless value is not accepted, since Rundeck reads 0
// as "the server's maximum" (30 days by default).
var authTokenDurationPattern = regexp.MustCompile(`^[1-9][0-9]*[smhdwy]$`)

// validAuthTokenDuration reports whether d is an acceptable
// auth_token_duration value, such as "30m", "1h" or "2d".
func validAuthTokenDuration(d string) bool {
	return authTokenDurationPattern.MatchString(d)
}

type TokenResp struct {
	User       string    `json:"user"`
	Token      string    `json:"token"`
	ID         string    `json:"id"`
	Creator    string    `json:"creator"`
	Name       string    `json:"name"`
	Expiration time.Time `json:"expiration"`
	Roles      []string  `json:"roles"`
	Expired    bool      `json:"expired"`
}

func getToken(d *schema.ResourceData) (string, error) {
	urlP, _ := d.Get("url").(string)
	username, _ := d.Get("auth_username").(string)
	password, _ := d.Get("auth_password").(string)

	return _getToken(urlP, username, password, "dev", DefaultAuthTokenDuration)

}

// listTokensForUser retrieves all tokens for a given user
// The client parameter should already have the User-Agent transport configured
func listTokensForUser(client *http.Client, urlP string, username string) ([]TokenResp, error) {
	tokenUrlString := fmt.Sprintf("%s/api/%s/tokens/%s", urlP, tokenAPIVersion, username)
	tokenUrl, err := url.Parse(tokenUrlString)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", tokenUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list tokens: statuscode %d\n%s", resp.StatusCode, string(body))
	}

	var tokens []TokenResp
	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// expiredTerraformTokens returns the tokens that the provider created for
// username in earlier runs and that Rundeck reports as expired. It relies on
// Rundeck's own expired flag rather than the local clock, so clock skew
// between Terraform and Rundeck cannot select a token that is still valid.
func expiredTerraformTokens(tokens []TokenResp, username string) []TokenResp {
	var expired []TokenResp
	for _, t := range tokens {
		if t.Name == terraformTokenName && t.User == username && t.Expired && t.ID != "" {
			expired = append(expired, t)
		}
	}
	return expired
}

// deleteToken deletes the API token with the given ID.
func deleteToken(client *http.Client, urlP string, id string) error {
	tokenUrlString := fmt.Sprintf("%s/api/%s/token/%s", urlP, tokenAPIVersion, url.PathEscape(id))
	req, err := http.NewRequest("DELETE", tokenUrlString, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("statuscode %d\n%s", resp.StatusCode, string(body))
	}
	return nil
}

// removeExpiredTerraformTokens deletes the expired tokens the provider created
// for username in earlier runs, so that creating a token on every run does not
// leave them piling up. Failures are logged and otherwise ignored: cleanup
// must never stop the provider from authenticating.
func removeExpiredTerraformTokens(client *http.Client, urlP string, username string) {
	tokens, err := listTokensForUser(client, urlP, username)
	if err != nil {
		log.Printf("[WARN] rundeck: could not list API tokens for %q to remove expired %s tokens: %s", username, terraformTokenName, err)
		return
	}
	for _, t := range expiredTerraformTokens(tokens, username) {
		if err := deleteToken(client, urlP, t.ID); err != nil {
			log.Printf("[WARN] rundeck: could not delete expired %s token %s: %s", terraformTokenName, t.ID, err)
		}
	}
}

// login authenticates username/password against Rundeck's form login. The
// session cookie it receives is kept in the client's cookie jar. Rundeck
// answers a rejected login by redirecting to /user/error (or back to
// /user/login) rather than with an error status, so the final URL is checked.
func login(client *http.Client, urlP string, username string, password string) error {
	data := url.Values{
		"j_username": {username},
		"j_password": {password},
	}
	resp, err := client.PostForm(fmt.Sprintf("%s/j_security_check", urlP), data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("login failed: statuscode %d", resp.StatusCode)
	}
	if p := resp.Request.URL.Path; strings.Contains(p, "/user/error") || strings.HasSuffix(p, "/user/login") {
		return fmt.Errorf("Rundeck rejected the login for user %q: check auth_username and auth_password (only local Rundeck accounts are supported)", username)
	}
	return nil
}

// createToken creates a new API token for username that expires after
// duration, and returns its value.
func createToken(client *http.Client, urlP string, username string, duration string) (string, error) {
	tokenBodyJson, err := json.Marshal(map[string]interface{}{
		"user":     username,
		"roles":    []string{"*"},
		"duration": duration,
		"name":     terraformTokenName,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/%s/tokens", urlP, tokenAPIVersion), bytes.NewBuffer(tokenBodyJson))
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("statuscode %d\n%s", resp.StatusCode, string(body))
	}

	tokenResp := &TokenResp{}
	if err := json.NewDecoder(resp.Body).Decode(tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Token == "" {
		return "", fmt.Errorf("Rundeck created API token %s but did not return its value", tokenResp.ID)
	}
	return tokenResp.Token, nil
}

// _getToken logs in with username/password and returns a new API token that
// expires after duration.
//
// A token is created on every call. Rundeck returns a token's value only in
// the response that creates it, never when tokens are listed, so a token
// created by an earlier run cannot be reused. Terraform also configures the
// provider separately for plan and for apply, so a single `terraform apply`
// creates two tokens. To keep them from accumulating, each token is
// short-lived (DefaultAuthTokenDuration unless auth_token_duration is set),
// and the user's expired terraform-token tokens are deleted before a new one
// is created.
func _getToken(urlP string, username string, password string, version string, duration string) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Jar:       jar,
		Transport: newUserAgentTransport(nil, version),
	}

	if err := login(client, urlP, username, password); err != nil {
		return "", err
	}

	removeExpiredTerraformTokens(client, urlP, username)

	return createToken(client, urlP, username, duration)
}
