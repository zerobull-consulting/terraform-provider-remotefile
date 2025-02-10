package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider/model"
	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider/retry"
	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider/sftp/connect"
	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider/sftp/parameters"
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
	resp.TypeName = "remotefile_sftp_contents"
}

// Schema defines the schema for the data source.
func (d *remoteFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a file from a remote system using SFTP.",
		Attributes: map[string]schema.Attribute{
			"allow_missing": schema.BoolAttribute{
				Description: "Whether to ignore that the file is missing",
				Optional:    true,
			},
			"contents": schema.StringAttribute{
				Description: "The file contents",
				Computed:    true,
				Sensitive:   true,
			},
			"host": schema.StringAttribute{
				Description: "The hostname",
				Required:    true,
			},
			"host_key": schema.StringAttribute{
				Description: "If set, the host key to verify against",
				Optional:    true,
			},
			"last_modified": schema.StringAttribute{
				Description: "The last modified timestamp",
				Computed:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password",
				Optional:    true,
				Sensitive:   true,
			},
			"path": schema.StringAttribute{
				Description: "The file path",
				Required:    true,
			},
			"port": schema.Int64Attribute{
				Description: "The port number",
				Optional:    true,
			},
			"private_key": schema.StringAttribute{
				Description: "The private key for connecting, PEM format (-----BEGIN OPENSSH PRIVATE KEY-----)",
				Optional:    true,
				Sensitive:   true,
			},
			"size": schema.Int64Attribute{
				Description: "The file size (in bytes)",
				Computed:    true,
			},
			"timeout": schema.StringAttribute{
				Description: "The connect timeout",
				Optional:    true,
			},
			"triggers": schema.MapAttribute{
				Description: "The triggers",
				Optional:    true,
				ElementType: types.StringType,
			},
			"user": schema.StringAttribute{
				Description: "The username",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the remote file",
				Computed:    true,
			},
			"retry_count": schema.Int64Attribute{
				Description: "Number of times to retry on failure",
				Optional:    true,
			},
			"retry_interval": schema.StringAttribute{
				Description: "Time to wait between retries (e.g. '10s')",
				Optional:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *remoteFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data model.RemoteFileDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set default retry values if not specified
	retryCount := int64(10)
	if !data.RetryCount.IsNull() {
		retryCount = data.RetryCount.ValueInt64()
	}

	retryInterval := 10 * time.Second
	if !data.RetryInterval.IsNull() {
		interval, err := time.ParseDuration(data.RetryInterval.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"invalid retry interval",
				fmt.Sprintf("enable to parse retry interval: %s", err),
			)
			return
		}
		retryInterval = interval
	}

	sshConnParams, err := parameters.CreateSSHConnectionParameters(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating SSH connection parameters",
			err.Error(),
		)
		return
	}

	operation := connect.ConnectAndCopy(sshConnParams, &data, &data)

	// Wrap the entire operation in the retry logic
	err = retry.WithRetry(retryCount, retryInterval, operation)

	if err != nil {
		resp.Diagnostics.AddError(
			"error reading remote file",
			err.Error(),
		)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
