package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/zerobull-consulting/terraform-provider-sftp/internal/provider/model"
	"github.com/zerobull-consulting/terraform-provider-sftp/internal/provider/ssh/connection/parameters"
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
	resp.TypeName = "sftp_remote_file"
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

func withRetry(retryCount int64, retryInterval time.Duration, operation func() error) error {
	var lastErr error

	for i := int64(0); i <= retryCount; i++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		if i == retryCount {
			return lastErr
		}

		time.Sleep(retryInterval)
	}

	return lastErr
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

	// Wrap the entire operation in the retry logic
	err = withRetry(retryCount, retryInterval, func() error {
		sshClient, err := ssh.Dial("tcp", sshConnParams.Address, sshConnParams.SshConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SSH server: %w", err)
		}
		defer sshClient.Close()

		// Create SFTP client
		sftpClient, err := sftp.NewClient(sshClient)
		if err != nil {
			return fmt.Errorf("error creating SFTP client: %w", err)
		}
		defer sftpClient.Close()

		// Get file info and contents
		fileInfo, err := sftpClient.Lstat(data.Path.ValueString())
		if err != nil {
			if data.AllowMissing.ValueBool() {
				data.ID = types.StringValue("missing")
				data.Contents = types.StringValue("")
				data.LastModified = types.StringValue(time.Now().Format(time.RFC3339))
				data.Size = types.Int64Value(-1)
				return nil
			}
			return fmt.Errorf("error reading remote file info: %w", err)
		}

		remoteFile, err := sftpClient.Open(data.Path.ValueString())
		if err != nil {
			return fmt.Errorf("error opening remote file: %w", err)
		}
		defer remoteFile.Close()

		buffer := bytes.NewBuffer(nil)
		_, err = io.Copy(buffer, remoteFile)
		if err != nil {
			return fmt.Errorf("error reading remote file contents: %w", err)
		}

		// Set model values
		data.ID = types.StringValue(fileInfo.Name())
		data.Contents = types.StringValue(buffer.String())
		data.LastModified = types.StringValue(fileInfo.ModTime().Format(time.RFC3339))
		data.Size = types.Int64Value(fileInfo.Size())

		return nil
	})

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
