package parameters

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

type SshConnectionParameters struct {
	SshConfig *ssh.ClientConfig
	Address   string
}

type SshModelSubset interface {
	GetHost() types.String
	GetHostKey() types.String
	GetPassword() types.String
	GetPrivateKey() types.String
	GetTimeout() types.String
	GetPort() types.Int64
	GetUser() types.String
}

func CreateSSHConnectionParameters(data SshModelSubset) (*SshConnectionParameters, error) {
	// Create a new SSH config based on the connection parameters from the data source model.
	if data.GetPassword().IsNull() && data.GetPrivateKey().IsNull() {
		return nil, errors.New("must provide either a password or private key")
	}

	var authMethod []ssh.AuthMethod

	if !data.GetPassword().IsNull() {
		authMethod = []ssh.AuthMethod{ssh.Password(data.GetPassword().ValueString())}
	} else {
		privateKeySigner, err := ssh.ParsePrivateKey([]byte(data.GetPrivateKey().ValueString()))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethod = []ssh.AuthMethod{ssh.PublicKeys(privateKeySigner)}
	}

	var hostKeyCallback ssh.HostKeyCallback
	if !data.GetHostKey().IsNull() {
		parsedHostKey, err := ssh.ParsePublicKey([]byte(data.GetHostKey().ValueString()))
		if err != nil {
			return nil, fmt.Errorf("failed to parse host key: %w", err)
		}
		hostKeyCallback = ssh.FixedHostKey(parsedHostKey)
	} else {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	timeout := "5m"
	if !data.GetTimeout().IsNull() {
		timeout = data.GetTimeout().ValueString()
	}
	timeoutDuration, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %w", err)
	}

	port := int64(22)
	if !data.GetPort().IsNull() {
		port = data.GetPort().ValueInt64()
	}

	sshConfig := &ssh.ClientConfig{
		User:            data.GetUser().ValueString(),
		Auth:            authMethod,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeoutDuration,
	}
	address := fmt.Sprintf("%s:%d", data.GetHost().ValueString(), port)

	return &SshConnectionParameters{
		SshConfig: sshConfig,
		Address:   address,
	}, nil
}
