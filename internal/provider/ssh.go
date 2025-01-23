package provider

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshConnectionParameters struct {
	sshConfig *ssh.ClientConfig
	address   string
}

func createSSHConnectionParameters(data *remoteFileDataSourceModel) (*sshConnectionParameters, error) {
	// Create a new SSH config based on the connection parameters from the data source model.
	if data.Password.IsNull() && data.PrivateKey.IsNull() {
		return nil, errors.New("must provide either a password or private key")
	}

	var authMethod []ssh.AuthMethod

	if !data.Password.IsNull() {
		authMethod = []ssh.AuthMethod{ssh.Password(data.Password.ValueString())}
	} else {
		privateKeySigner, err := ssh.ParsePrivateKey([]byte(data.PrivateKey.ValueString()))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethod = []ssh.AuthMethod{ssh.PublicKeys(privateKeySigner)}
	}

	var hostKeyCallback ssh.HostKeyCallback
	if !data.HostKey.IsNull() {
		parsedHostKey, err := ssh.ParsePublicKey([]byte(data.HostKey.ValueString()))
		if err != nil {
			return nil, fmt.Errorf("failed to parse host key: %w", err)
		}
		hostKeyCallback = ssh.FixedHostKey(parsedHostKey)
	} else {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	timeout := "5m"
	if !data.Timeout.IsNull() {
		timeout = data.Timeout.ValueString()
	}
	timeoutDuration, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %w", err)
	}

	port := int64(22)
	if !data.Port.IsNull() {
		port = data.Port.ValueInt64()
	}

	sshConfig := &ssh.ClientConfig{
		User:            data.User.ValueString(),
		Auth:            authMethod,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeoutDuration,
	}
	address := fmt.Sprintf("%s:%d", data.Host.ValueString(), port)

	return &sshConnectionParameters{
		sshConfig: sshConfig,
		address:   address,
	}, nil
}
