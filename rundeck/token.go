package rundeck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

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
	apiVersion, _ := d.Get("api_version").(string)
	username, _ := d.Get("auth_username").(string)
	password, _ := d.Get("auth_password").(string)

	return _getToken(urlP, apiVersion, username, password, "dev")

}

// listTokensForUser retrieves all tokens for a given user
// The client parameter should already have the User-Agent transport configured
func listTokensForUser(client *http.Client, urlP string, username string) ([]TokenResp, error) {
	// Use API v43 for token listing (same as token creation)
	tokenUrlString := fmt.Sprintf("%s/api/%s/tokens/%s", urlP, "43", username)
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

// findValidTerraformToken looks for an existing valid terraform-token
// If multiple valid tokens exist (e.g., from previous provider versions with the bug),
// returns the first one found. Users can manually clean up duplicate tokens if desired.
func findValidTerraformToken(tokens []TokenResp) *TokenResp {
	now := time.Now()
	for i := range tokens {
		token := &tokens[i]
		// Check if this is a terraform-token and it's not expired
		if token.Name == "terraform-token" && !token.Expired {
			// For tokens with duration "0" (never expire), Expiration might be zero or far future
			if token.Expiration.IsZero() || token.Expiration.After(now) {
				return token
			}
		}
	}
	return nil
}

func _getToken(urlP string, apiVersion string, username string, password string, version string) (string, error) {

	secCheckUrlString := fmt.Sprintf("%s/j_security_check", urlP)
	secCheckUrl, err := url.Parse(secCheckUrlString)

	if err != nil {
		return "", err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Jar:       jar,
		Transport: newUserAgentTransport(nil, version),
	}

	data := url.Values{
		"j_username": {username},
		"j_password": {password},
	}
	_, err = client.PostForm(secCheckUrl.String(), data)
	if err != nil {
		return "", err
	}

	// First, check if a valid terraform-token already exists
	existingTokens, err := listTokensForUser(client, urlP, username)
	if err == nil {
		// If we successfully listed tokens, check for existing valid terraform-token
		if existingToken := findValidTerraformToken(existingTokens); existingToken != nil {
			// Return the existing token value
			if version, _ := strconv.Atoi(apiVersion); version >= 19 {
				return existingToken.Token, nil
			} else if existingToken.Token != "" {
				return existingToken.Token, nil
			} else if existingToken.ID != "" {
				return existingToken.ID, nil
			}
		}
	}
	// If we couldn't list tokens or no valid token exists, create a new one

	tokenUrlString := fmt.Sprintf("%s/api/%s/tokens", urlP, "43")
	tokenUrl, err := url.Parse(tokenUrlString)

	if err != nil {
		return "", err
	}

	tokenBody := map[string]interface{}{}
	tokenBody["user"] = username
	tokenBody["roles"] = []string{"*"}
	tokenBody["duration"] = "0"
	tokenBody["name"] = "terraform-token"
	tokenBodyJson, err := json.Marshal(tokenBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", tokenUrl.String(), bytes.NewBuffer(tokenBodyJson))
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("statuscode %d\n%s", resp.StatusCode, string(body))
	} else if err != nil {
		return "", err
	}

	tokenResp := &TokenResp{}
	err = json.NewDecoder(resp.Body).Decode(tokenResp)
	if err != nil {
		return "", err
	}

	if version, _ := strconv.Atoi(apiVersion); version >= 19 {
		return tokenResp.Token, nil
	} else if tokenResp.Token != "" {
		return tokenResp.Token, nil
	} else if tokenResp.ID != "" {
		return tokenResp.ID, nil
	}

	return tokenResp.Token, nil

}
