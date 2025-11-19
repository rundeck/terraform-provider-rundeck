package rundeck

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/rundeck/go-rundeck/rundeck"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
	"github.com/rundeck/go-rundeck/rundeck/auth"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_URL", nil),
				Description: "URL of the root of the target Rundeck server.",
			},
			"api_version": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_API_VERSION", "46"),
				Description: "API Version of the target Rundeck server (minimum: 46 for Rundeck 5.0.0+).",
			},
			"auth_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_AUTH_TOKEN", nil),
				Description: "Auth token to use with the Rundeck API.",
			},
			"auth_username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_AUTH_USERNAME", nil),
				Description: "Username used to request a token for the Rundeck API.",
			},
			"auth_password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_AUTH_PASSWORD", nil),
				Description: "Password used to request a token for the Rundeck API.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			// "rundeck_job": resourceRundeckJob(), // Migrated to Framework with JSON support
			// "rundeck_project":        resourceRundeckProject(), // Migrated to Framework
			// "rundeck_private_key":    resourceRundeckPrivateKey(), // Migrated to Framework
			// "rundeck_password":       resourceRundeckPassword(), // Migrated to Framework
			// "rundeck_public_key":     resourceRundeckPublicKey(), // Migrated to Framework
			// "rundeck_acl_policy":     resourceRundeckAclPolicy(), // Migrated to Framework
			// "rundeck_system_runner":  resourceRundeckSystemRunner(), // Migrated to Framework
			// "rundeck_project_runner": resourceRundeckProjectRunner(), // Migrated to Framework
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	urlP, _ := d.Get("url").(string)
	apiVersion, _ := d.Get("api_version").(string)

	apiURLString := fmt.Sprintf("%s/api/%s", urlP, apiVersion)

	apiURL, error := url.Parse(apiURLString)

	if error != nil {
		return nil, error
	}

	_, okToken := d.GetOk("auth_token")
	_, okUsername := d.GetOk("auth_username")
	_, okPassword := d.GetOk("auth_password")

	var token string

	if okToken {
		token = d.Get("auth_token").(string)
	} else if okUsername && okPassword {
		t, err := getToken(d)
		token = t
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("auth_token need to be set of auth_username and auth_password")
	}

	// Create the original v1 client
	clientV1 := rundeck.NewRundeckWithBaseURI(apiURL.String())
	clientV1.Authorizer = &auth.TokenAuthorizer{Token: token}

	// Create the new v2 client
	cfg := openapi.NewConfiguration()
	cfg.Host = apiURL.Host
	cfg.Scheme = apiURL.Scheme

	clientV2 := openapi.NewAPIClient(cfg)

	// Create a context with the API token as a header
	ctx := context.WithValue(context.Background(), openapi.ContextAPIKeys, map[string]openapi.APIKey{
		"rundeckApiToken": {
			Key: token,
		},
	})

	return &RundeckClients{
		V1:    &clientV1,
		V2:    clientV2,
		Token: token,
		ctx:   ctx,
	}, nil
}

// RundeckClients wraps both v1 and v2 clients
type RundeckClients struct {
	V1         *rundeck.BaseClient
	V2         *openapi.APIClient
	Token      string
	BaseURL    string // Base Rundeck URL without /api/version
	APIVersion string
	ctx        context.Context
}
