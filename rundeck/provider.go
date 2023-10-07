package rundeck

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

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
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_API_VERSION", "14"),
				Description: "API Version of the target Rundeck server.",
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
				Description: "Username used to request a tiken for the Rundeck API.",
			},
			"auth_password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_AUTH_PASSWORD", nil),
				Description: "Password used to request a tiken for the Rundeck API.",
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

	client := rundeck.NewRundeckWithBaseURI(apiURL.String())
	client.Authorizer = &auth.TokenAuthorizer{Token: token}

	return &client, nil
}
