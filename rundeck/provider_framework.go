package rundeck

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rundeck/go-rundeck/rundeck"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
	"github.com/rundeck/go-rundeck/rundeck/auth"
)

// Ensure the implementation satisfies the provider.Provider interface
var _ provider.Provider = &frameworkProvider{}

// frameworkProvider is the provider implementation.
type frameworkProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance testing.
	version string
}

// frameworkProviderModel describes the provider data model.
type frameworkProviderModel struct {
	URL          types.String `tfsdk:"url"`
	APIVersion   types.String `tfsdk:"api_version"`
	AuthToken    types.String `tfsdk:"auth_token"`
	AuthUsername types.String `tfsdk:"auth_username"`
	AuthPassword types.String `tfsdk:"auth_password"`
}

func NewFrameworkProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &frameworkProvider{
			version: version,
		}
	}
}

func (p *frameworkProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "rundeck"
	resp.Version = p.version
}

func (p *frameworkProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "URL of the root of the target Rundeck server.",
				Optional:    true,
			},
			"api_version": schema.StringAttribute{
				Description: "API Version of the target Rundeck server (minimum: 46 for Rundeck 5.0.0+).",
				Optional:    true,
			},
			"auth_token": schema.StringAttribute{
				Description: "Auth token to use with the Rundeck API.",
				Optional:    true,
			},
			"auth_username": schema.StringAttribute{
				Description: "Username used to request a token for the Rundeck API.",
				Optional:    true,
			},
			"auth_password": schema.StringAttribute{
				Description: "Password used to request a token for the Rundeck API.",
				Optional:    true,
			},
		},
	}
}

func (p *frameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config frameworkProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get configuration values with environment variable fallbacks (matching SDK behavior)
	urlString := config.URL.ValueString()
	if urlString == "" {
		urlString = os.Getenv("RUNDECK_URL")
	}
	if urlString == "" {
		resp.Diagnostics.AddError(
			"Missing Rundeck URL",
			"The provider requires a Rundeck URL to be configured via the url attribute or RUNDECK_URL environment variable",
		)
		return
	}

	apiVersion := config.APIVersion.ValueString()
	if apiVersion == "" {
		apiVersion = os.Getenv("RUNDECK_API_VERSION")
	}
	if apiVersion == "" {
		apiVersion = "46" // Default API version - Rundeck 5.0.0+
	}

	authToken := config.AuthToken.ValueString()
	if authToken == "" {
		authToken = os.Getenv("RUNDECK_AUTH_TOKEN")
	}

	authUsername := config.AuthUsername.ValueString()
	if authUsername == "" {
		authUsername = os.Getenv("RUNDECK_AUTH_USERNAME")
	}

	authPassword := config.AuthPassword.ValueString()
	if authPassword == "" {
		authPassword = os.Getenv("RUNDECK_AUTH_PASSWORD")
	}

	// Determine authentication method
	var token string
	if authToken != "" {
		token = authToken
	} else if authUsername != "" && authPassword != "" {
		// Token generation from username/password would need to be implemented here
		// For now, we'll require auth_token
		resp.Diagnostics.AddError(
			"Authentication Method Not Yet Supported",
			"Username/password authentication is not yet implemented in the framework provider. Please use auth_token.",
		)
		return
	} else {
		resp.Diagnostics.AddError(
			"Missing Authentication",
			"Either auth_token or both auth_username and auth_password must be provided",
		)
		return
	}

	// Create Rundeck clients (reusing the same structure as SDK provider)
	apiURLString := fmt.Sprintf("%s/api/%s", urlString, apiVersion)
	apiURL, err := url.Parse(apiURLString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid URL",
			fmt.Sprintf("Unable to parse Rundeck URL: %s", err.Error()),
		)
		return
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
	ctxWithAuth := context.WithValue(context.Background(), openapi.ContextAPIKeys, map[string]openapi.APIKey{
		"rundeckApiToken": {
			Key: token,
		},
	})

	clients := &RundeckClients{
		V1:         &clientV1,
		V2:         clientV2,
		Token:      token,
		BaseURL:    fmt.Sprintf("%s://%s", apiURL.Scheme, apiURL.Host),
		APIVersion: apiVersion,
		ctx:        ctxWithAuth,
	}

	resp.DataSourceData = clients
	resp.ResourceData = clients
}

func (p *frameworkProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAclPolicyResource,
		NewPrivateKeyResource,
		NewPublicKeyResource,
		NewPasswordResource,
		NewSystemRunnerResource,
		NewProjectRunnerResource,
		NewProjectResource,
		NewJobResource,
	}
}

func (p *frameworkProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Data sources will be added here if needed
	}
}
