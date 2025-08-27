package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider/model"
	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider/retry"
	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider/sftp/connect"
	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider/sftp/parameters"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource              = &remoteFileResource{}
	_ resource.ResourceWithConfigure = &remoteFileResource{}
)

// NewRemoteFileResource is a helper function to simplify the provider implementation
func NewRemoteFileResource() resource.Resource {
	return &remoteFileResource{}
}

// remoteFileResource is the resource implementation
type remoteFileResource struct{}

// Configure adds the provider configured client to the resource.
func (r *remoteFileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Nothing to configure
}

// Metadata returns the resource type name
func (r *remoteFileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sftp"
}

// Schema defines the schema for the resource
func (r *remoteFileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a file on a remote system using SFTP.",
		Attributes: map[string]schema.Attribute{
			"contents": schema.StringAttribute{
				Description: "The file contents",
				Required:    true,
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.StringAttribute{
				Description: "The file permissions (e.g. '0644')",
				Optional:    true,
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

// Create creates the resource and sets the initial Terraform state
func (r *remoteFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data model.RemoteFileDataSourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
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
				fmt.Sprintf("unable to parse retry interval: %s", err),
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

	// Write the file to the remote server
	operation := connect.ConnectAndWrite(sshConnParams, &data, &data)

	// Wrap the entire operation in the retry logic
	err = retry.WithRetry(retryCount, retryInterval, operation)
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating remote file",
			err.Error(),
		)
		return
	}

	// Generate an ID for the resource
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Host.ValueString(), data.Path.ValueString()))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data
func (r *remoteFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform prior state data into the model
	var data model.RemoteFileDataSourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
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
				fmt.Sprintf("unable to parse retry interval: %s", err),
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

	// Read the file from the remote server
	operation := connect.ConnectAndCopy(sshConnParams, &data, &data)

	// Wrap the entire operation in the retry logic
	err = retry.WithRetry(retryCount, retryInterval, operation)
	if err != nil {
		if connect.IsFileNotFound(err) {
			resp.Diagnostics.AddWarning(
				"remote file not found",
				fmt.Sprintf("remote file %s not found, removing from state", data.Path.ValueString()),
			)
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"error reading remote file",
			err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success
func (r *remoteFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data model.RemoteFileDataSourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
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
				fmt.Sprintf("unable to parse retry interval: %s", err),
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

	// Write the file to the remote server
	operation := connect.ConnectAndWrite(sshConnParams, &data, &data)

	// Wrap the entire operation in the retry logic
	err = retry.WithRetry(retryCount, retryInterval, operation)
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating remote file",
			err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success
func (r *remoteFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data model.RemoteFileDataSourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
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
				fmt.Sprintf("unable to parse retry interval: %s", err),
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

	// Delete the file from the remote server
	operation := connect.ConnectAndDelete(sshConnParams, &data)

	// Wrap the entire operation in the retry logic
	err = retry.WithRetry(retryCount, retryInterval, operation)
	if err != nil {
		// If the file doesn't exist, that's okay - we're deleting it anyway
		if !connect.IsFileNotFound(err) {
			resp.Diagnostics.AddError(
				"error deleting remote file",
				err.Error(),
			)
			return
		}
	}
}

func (r *remoteFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Split the ID by : to get host and path
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'host:path'",
		)
		return
	}

	host := parts[0]
	filePath := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("host"), host)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), filePath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}