package rundeck

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	// "github.com/apparentlymart/go-rundeck-api/rundeck"
	"github.com/rundeck/go-rundeck/rundeck"
	"github.com/rundeck/go-rundeck/rundeck/auth"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_URL", nil),
				Description: "URL of the root of the target Rundeck server.",
			},
			"api_version": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_API_VERSION", nil),
				Description: "API Version of the target Rundeck server.",
			},
			"auth_token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_AUTH_TOKEN", nil),
				Description: "Auth token to use with the Rundeck API.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"rundeck_project":     resourceRundeckProject(),
			"rundeck_job":         resourceRundeckJob(),
			"rundeck_private_key": resourceRundeckPrivateKey(),
			"rundeck_public_key":  resourceRundeckPublicKey(),
			"rundeck_acl_policy":  resourceRundeckAclPolicy(),
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

	token := d.Get("auth_token").(string)

	client := rundeck.NewRundeckWithBaseURI(apiURL.String())
	client.Authorizer = &auth.TokenAuthorizer{Token: token}

	return &client, nil
}
