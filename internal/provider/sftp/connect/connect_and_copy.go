package connect

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SshConnectionParameters interface {
	GetSshConfig() *ssh.ClientConfig
	GetAddress() string
}

type InputModel interface {
	GetPath() types.String
	GetAllowMissing() types.Bool
}

type OutputModel interface {
	SetID(types.String)
	GetID() types.String
	SetContents(types.String)
	GetContents() types.String
	SetLastModified(types.String)
	GetLastModified() types.String
	SetSize(types.Int64)
	GetSize() types.Int64
}

// private interfaces for ssh and sftp clients. allows swapping the implementation if needed
type sshDialer interface {
	Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
}

// Create a default dialer type
type defaultSshDialer struct{}

// Implement the sshDialer interface for defaultSshDialer
func (d *defaultSshDialer) Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	return ssh.Dial(network, addr, config)
}

type sftpCreator interface {
	NewClient(conn *ssh.Client) (*sftp.Client, error)
}

// create a default sftp type
type defaultSftpCreator struct{}

// Implement the sftpCreator interface for defaultSftpCreator
func (d *defaultSftpCreator) NewClient(conn *ssh.Client) (*sftp.Client, error) {
	return sftp.NewClient(conn)
}

func ConnectAndCopy(sshConnParams SshConnectionParameters, input InputModel, output OutputModel) func() error {
	sshDialer := &defaultSshDialer{}
	sftpClient := &defaultSftpCreator{}
	return connectAndCopyOperationWithDeps(sshConnParams, input, output, sshDialer, sftpClient)
}

func connectAndCopyOperationWithDeps(sshConnParams SshConnectionParameters, input InputModel, output OutputModel, sshDial sshDialer, sftpNew sftpCreator) func() error {
	return func() error {
		sshClient, err := sshDial.Dial("tcp", sshConnParams.GetAddress(), sshConnParams.GetSshConfig())
		if err != nil {
			return fmt.Errorf("failed to connect to SSH server: %w", err)
		}
		defer sshClient.Close()

		sftpClient, err := sftpNew.NewClient(sshClient)
		if err != nil {
			return fmt.Errorf("error creating SFTP client: %w", err)
		}
		defer sftpClient.Close()

		// Get file info and contents
		fileInfo, err := sftpClient.Lstat(input.GetPath().ValueString())
		if err != nil {
			if input.GetAllowMissing().ValueBool() {
				output.SetID(types.StringValue("missing"))
				output.SetContents(types.StringValue(""))
				output.SetLastModified(types.StringValue(time.Now().Format(time.RFC3339)))
				output.SetSize(types.Int64Value(-1))
				return nil
			}
			return fmt.Errorf("error reading remote file info: %w", err)
		}

		remoteFile, err := sftpClient.Open(input.GetPath().ValueString())
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
		output.SetID(types.StringValue(fileInfo.Name()))
		output.SetContents(types.StringValue(buffer.String()))
		output.SetLastModified(types.StringValue(fileInfo.ModTime().Format(time.RFC3339)))
		output.SetSize(types.Int64Value(fileInfo.Size()))

		return nil
	}
}
