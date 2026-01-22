package rundeck

import (
	"os"
	"testing"
)

func TestAccToken(t *testing.T) {
	testAccPreCheck(t)

	username := os.Getenv("RUNDECK_AUTH_USERNAME")
	password := os.Getenv("RUNDECK_AUTH_PASSWORD")
	apiVersion := os.Getenv("RUNDECK_API_VERSION")
	if apiVersion == "" {
		apiVersion = "14"
	}
	url := os.Getenv("RUNDECK_URL")

	if username != "" && password != "" {
		_, err := _getToken(url, apiVersion, username, password)
		if err != nil {
			t.Fatalf("failed to get a token: %s", err)
		}
	} else {
		t.Logf("Can not run TestAccToken test as there is no username and password defined")
	}
}

func TestAccTokenReuse(t *testing.T) {
	testAccPreCheck(t)

	username := os.Getenv("RUNDECK_AUTH_USERNAME")
	password := os.Getenv("RUNDECK_AUTH_PASSWORD")
	apiVersion := os.Getenv("RUNDECK_API_VERSION")
	if apiVersion == "" {
		apiVersion = "14"
	}
	url := os.Getenv("RUNDECK_URL")

	if username == "" || password == "" {
		t.Skip("Skipping TestAccTokenReuse: RUNDECK_AUTH_USERNAME and RUNDECK_AUTH_PASSWORD not set")
	}

	// Get token first time - this may create a new token or reuse existing
	token1, err := _getToken(url, apiVersion, username, password)
	if err != nil {
		t.Fatalf("failed to get token first time: %s", err)
	}

	if token1 == "" {
		t.Fatal("first token is empty")
	}

	// Get token second time - should reuse the existing token
	token2, err := _getToken(url, apiVersion, username, password)
	if err != nil {
		t.Fatalf("failed to get token second time: %s", err)
	}

	if token2 == "" {
		t.Fatal("second token is empty")
	}

	// Verify the same token is returned (reused, not recreated)
	if token1 != token2 {
		t.Errorf("Expected token to be reused, but got different tokens:\nFirst:  %s\nSecond: %s", token1, token2)
	}

	t.Logf("Successfully verified token reuse: %s", token1)
}
