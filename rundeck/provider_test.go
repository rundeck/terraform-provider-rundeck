package rundeck

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"

	"github.com/rundeck/go-rundeck/rundeck"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
	"github.com/rundeck/go-rundeck/rundeck/auth"
)

// To run these acceptance tests, you will need a Rundeck server.
// An easy way to get one is to use Rundeck's "Anvils" demo, which includes a Vagrantfile
// to get it running easily:
//    https://github.com/rundeck/anvils-demo
// The anvils demo ships with some example security policies that don't have enough access to
// run the tests, so you need to either modify one of the stock users to have full access or
// create a new user with such access. The following block is an example that gives the
// 'admin' user and API clients open access.
// In the anvils demo the admin password is "admin" by default.

// Place the contents of the following comment in /etc/rundeck/terraform-test.aclpolicy
/*
description: Admin, all access.
context:
  project: '.*' # all projects
for:
  resource:
    - allow: '*' # allow read/create all kinds
  adhoc:
    - allow: '*' # allow read/running/killing adhoc jobs
  job:
    - allow: '*' # allow read/write/delete/run/kill of all jobs
  node:
    - allow: '*' # allow read/run for all nodes
by:
  group: admin
---
description: Admin, all access.
context:
  application: 'rundeck'
for:
  resource:
    - allow: '*' # allow create of projects
  project:
    - allow: '*' # allow view/admin of all projects
  storage:
    - allow: '*' # allow read/create/update/delete for all /keys/* storage content
by:
  group: admin
---
description: Admin API, all access.
context:
  application: 'rundeck'
for:
  resource:
    - allow: '*' # allow create of projects
  project:
    - allow: '*' # allow view/admin of all projects
  storage:
    - allow: '*' # allow read/create/update/delete for all /keys/* storage content
by:
  group: api_token_group
*/

// Once you've got a user set up, put that user's API auth token in the RUNDECK_AUTH_TOKEN
// environment variable, and put the URL of the Rundeck home page in the RUNDECK_URL variable.
// If you're using the Anvils demo in its default configuration, you can find or generate an API
// token at http://192.168.50.2:4440/user/profile once you've logged in, and RUNDECK_URL will
// be http://192.168.50.2:4440/ .

func testAccProtoV5ProviderFactories() map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){
		"rundeck": func() (tfprotov5.ProviderServer, error) {
			// Use Framework provider directly (SDKv2 removed)
			return providerserver.NewProtocol5(NewFrameworkProvider("test")())(), nil
		},
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("RUNDECK_URL"); v == "" {
		t.Fatal("RUNDECK_URL must be set for acceptance tests")
	}

	token := os.Getenv("RUNDECK_AUTH_TOKEN")
	username := os.Getenv("RUNDECK_AUTH_USERNAME")
	password := os.Getenv("RUNDECK_AUTH_PASSWORD")
	if !(token != "" || (username != "" && password != "")) {
		t.Logf("RUNDECK_AUTH_TOKEN=%s", token)
		t.Logf("RUNDECK_AUTH_USERNAME=%s", username)
		t.Logf("RUNDECK_AUTH_PASSWORD=%s", password)
		t.Fatal("RUNDECK_AUTH_TOKEN must be set for acceptance tests or RUNDECK_AUTH_USERNAME and RUNDECK_AUTH_PASSWORD")
	}
}

// getTestClients creates Rundeck clients from environment variables for test verification
func getTestClients() (*RundeckClients, error) {
	urlP := os.Getenv("RUNDECK_URL")
	if urlP == "" {
		return nil, fmt.Errorf("RUNDECK_URL must be set")
	}

	apiVersion := os.Getenv("RUNDECK_API_VERSION")
	if apiVersion == "" {
		apiVersion = "56" // Default to v56 (Rundeck 5.17.0+) for full feature support including runners
	}

	token := os.Getenv("RUNDECK_AUTH_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("RUNDECK_AUTH_TOKEN must be set")
	}

	apiURLString := fmt.Sprintf("%s/api/%s", urlP, apiVersion)
	apiURL, err := url.Parse(apiURLString)
	if err != nil {
		return nil, err
	}

	// Create the V1 client
	clientV1 := rundeck.NewRundeckWithBaseURI(apiURL.String())
	clientV1.Authorizer = &auth.TokenAuthorizer{Token: token}

	// Create the V2 client
	cfg := openapi.NewConfiguration()
	cfg.Host = apiURL.Host
	cfg.Scheme = apiURL.Scheme

	clientV2 := openapi.NewAPIClient(cfg)

	// Create a context with the API token
	ctx := context.WithValue(context.Background(), openapi.ContextAPIKeys, map[string]openapi.APIKey{
		"rundeckApiToken": {
			Key: token,
		},
	})

	return &RundeckClients{
		V1:         &clientV1,
		V2:         clientV2,
		Token:      token,
		BaseURL:    urlP,
		APIVersion: apiVersion,
		ctx:        ctx,
	}, nil
}
