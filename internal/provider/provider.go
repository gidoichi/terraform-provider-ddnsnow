package provider

import (
	"context"

	"terraform-provider-ddnsnow/pkg/ddnsnow"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &ddnsnowProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ddnsnowProvider{
			version: version,
		}
	}
}

// ddnsnowProvider is the provider implementation.
type ddnsnowProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ddnsnowProviderModel maps provider schema data to a Go type.
type ddnsnowProviderModel struct {
	Username     types.String `tfsdk:"username"`
	PasswordHash types.String `tfsdk:"password_hash"`
	Server       types.String `tfsdk:"server"`
}

// Metadata returns the provider type name.
func (p *ddnsnowProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ddnsnow"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *ddnsnowProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Optional: true,
			},
			"password_hash": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"server": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

// Configure prepares a DDNS Now API client for data sources and resources.
func (p *ddnsnowProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config ddnsnowProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown DDNS Now API Username",
			"The provider cannot create the DDNS Now API client as there is an unknown configuration value for the DDNS Now username. "+
				"Target apply the source of the value first, set the value statically in the configuration.",
		)
	}

	if config.PasswordHash.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password_hash"),
			"Unknown DDNS Now Password Hash",
			"The provider cannot create the DDNS Now API client as there is an unknown configuration value for the DDNS Now Password Hash. "+
				"Target apply the source of the value first, set the value statically in the configuration.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var username, passwordHash, server string

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.PasswordHash.IsNull() {
		passwordHash = config.PasswordHash.ValueString()
	}

	if !config.Server.IsNull() {
		server = config.Server.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing DDNS Now Username",
			"The provider cannot create the DDNS Now API client as there is a missing or empty value for the DDNS Now username. "+
				"Set the username value in the configuration. "+
				"If this is already set, ensure the value is not empty.",
		)
	}

	if passwordHash == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password_hash"),
			"Missing DDNS Now Password Hash",
			"The provider cannot create the DDNS Now API client as there is a missing or empty value for the DDNS Now Password Hash. "+
				"Set the password_hash value in the configuration. "+
				"If this is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new DDNS Now client using the configuration values
	client, err := ddnsnow.NewClient(&username, &passwordHash, &server)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create DDNS Now API Client",
			"An unexpected error occurred when creating the DDNS Now API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"DDNS Now Client Error: "+err.Error(),
		)
		return
	}

	// Make the DDNS Now client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *ddnsnowProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *ddnsnowProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRecordResource,
	}
}
