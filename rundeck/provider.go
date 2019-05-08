package rundeck

import (
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
			"auth_token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_AUTH_TOKEN", nil),
				Description: "Auth token to use with the Rundeck API.",
			},
			"allow_unverified_ssl": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If set, the Rundeck client will permit unverifiable SSL certificates.",
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
	apiURL, error := url.Parse(d.Get("url").(string))

	if error != nil {
		return nil, error
	}

	token := d.Get("auth_token").(string)

	client := rundeck.NewRundeckWithBaseURI(apiURL.String())
	client.Authorizer = &auth.TokenAuthorizer{Token: token}

	return &client, nil
}
