package connect

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// DeleteInputModel interface defines the methods required for deleting a remote file
type DeleteInputModel interface {
	GetPath() types.String
}

// IsFileNotFound checks if the error is related to file not found
func IsFileNotFound(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for os.IsNotExist or look for common error messages
	return os.IsNotExist(err) || 
		   contains(err.Error(), "no such file") || 
		   contains(err.Error(), "file does not exist")
}

// Helper function to check if a string contains a substring
	return strings.Contains(strings.ToLower(s), substr)
}

// ConnectAndDelete creates an operation to delete a file from a remote server
func ConnectAndDelete(sshConnParams SshConnectionParameters, input DeleteInputModel) func() error {
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

		// Delete the file
		err = sftpClient.Remove(input.GetPath().ValueString())
		if err != nil {
			// Check if the file is already gone
			if IsFileNotFound(err) {
				// File doesn't exist, that's fine for a delete operation
				return nil
			}
			return fmt.Errorf("error deleting remote file: %w", err)
		}

		return nil
	}
}
