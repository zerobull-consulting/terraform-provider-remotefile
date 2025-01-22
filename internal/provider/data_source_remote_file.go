package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &remoteFileDataSource{}
)

// NewRemoteFileDataSource is a helper function to simplify the provider implementation.
func NewRemoteFileDataSource() datasource.DataSource {
	return &remoteFileDataSource{}
}

// remoteFileDataSource is the data source implementation.
type remoteFileDataSource struct{}

// Metadata returns the data source type name.
func (d *remoteFileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_file"
}

// Schema defines the schema for the data source.
func (d *remoteFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

// Read refreshes the Terraform state with the latest data.
func (d *remoteFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
}
