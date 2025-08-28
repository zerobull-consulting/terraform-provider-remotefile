package connect

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// WriteInputModel interface defines the methods required for writing to a remote file
type WriteInputModel interface {
	GetPath() types.String
	GetContents() types.String
	GetPermissions() types.String
}

// ConnectAndWrite creates an operation to write file content to a remote server
func ConnectAndWrite(sshConnParams SshConnectionParameters, input WriteInputModel, output OutputModel) func() error {
	return func() error {
		sshClient, err := ssh.Dial("tcp", sshConnParams.GetAddress(), sshConnParams.GetSshConfig())
		if err != nil {
			return fmt.Errorf("failed to connect to SSH server: %w", err)
		}
		defer sshClient.Close()

		sftpClient, err := sftp.NewClient(sshClient)
		if err != nil {
			return fmt.Errorf("error creating SFTP client: %w", err)
		}
		defer sftpClient.Close()

		// Create or overwrite the file
		remoteFile, err := sftpClient.Create(input.GetPath().ValueString())
		if err != nil {
			return fmt.Errorf("error creating remote file: %w", err)
		}
		defer remoteFile.Close()

		// Write the file contents
		contentBytes := []byte(input.GetContents().ValueString())
		_, err = remoteFile.Write(contentBytes)
		if err != nil {
			return fmt.Errorf("error writing to remote file: %w", err)
		}

		// Set permissions if specified
		if !input.GetPermissions().IsNull() && input.GetPermissions().ValueString() != "" {
			// Parse permission string (e.g., "0644") to os.FileMode
			modeStr := input.GetPermissions().ValueString()
			modeInt, err := strconv.ParseUint(modeStr, 8, 32)
			if err != nil {
				return fmt.Errorf("error parsing file permissions %s: %w", modeStr, err)
			}
			mode := os.FileMode(modeInt)

			err = sftpClient.Chmod(input.GetPath().ValueString(), mode)
			if err != nil {
				return fmt.Errorf("error setting file permissions: %w", err)
			}
		}

		// Get updated file info
		fileInfo, err := sftpClient.Lstat(input.GetPath().ValueString())
		if err != nil {
			return fmt.Errorf("error reading remote file info after write: %w", err)
		}

		// Set model values
		output.SetID(types.StringValue(fileInfo.Name()))
		output.SetContents(types.StringValue(string(contentBytes)))
		output.SetLastModified(types.StringValue(fileInfo.ModTime().Format(time.RFC3339)))
		output.SetSize(types.Int64Value(fileInfo.Size()))

		return nil
	}
}
