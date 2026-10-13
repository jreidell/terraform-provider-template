// Package provider implements the template Terraform provider.
package provider

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jreidell/terraform-provider-template/internal/uuidclient"
)

// EndpointEnvVar is read when the provider block does not set endpoint.
const EndpointEnvVar = "TEMPLATE_ENDPOINT"

const requestTimeout = 30 * time.Second

var _ provider.Provider = &TemplateProvider{}

// TemplateProvider is the provider implementation.
type TemplateProvider struct {
	// version is "dev" for local builds and "test" under acceptance testing.
	version string
}

// TemplateProviderModel maps the provider block to Go types.
type TemplateProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
}

// New returns a factory for the provider, as providerserver expects.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TemplateProvider{version: version}
	}
}

func (p *TemplateProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	// TypeName is the prefix of every resource and data source: template_*.
	resp.TypeName = "template"
	resp.Version = p.version
}

func (p *TemplateProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The template provider is a starting point for building Terraform providers " +
			"with the Terraform Plugin Framework.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the UUID generation API. Can also be set with the `" +
					EndpointEnvVar + "` environment variable. Defaults to `" + uuidclient.DefaultEndpoint + "`.",
				Optional: true,
			},
		},
	}
}

func (p *TemplateProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data TemplateProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The client is built once here, so a value that is only known after
	// apply cannot be used.
	if data.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown API Endpoint",
			"The provider cannot create the API client because the endpoint is not known until apply. "+
				"Set the value statically in the configuration or use the "+EndpointEnvVar+" environment variable.",
		)

		return
	}

	// Precedence: configuration, then environment, then the default.
	endpoint := os.Getenv(EndpointEnvVar)

	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.ValueString()
	}

	if endpoint == "" {
		endpoint = uuidclient.DefaultEndpoint
	}

	client := uuidclient.New(
		endpoint,
		&http.Client{Timeout: requestTimeout},
		"terraform-provider-template/"+p.version,
	)

	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *TemplateProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServerResource,
	}
}

func (p *TemplateProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
