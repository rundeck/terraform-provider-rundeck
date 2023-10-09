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
