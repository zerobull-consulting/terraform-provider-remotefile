package parameters

import "golang.org/x/crypto/ssh"

type SshConnectionParameters struct {
	sshConfig *ssh.ClientConfig
	address   string
}

func (s *SshConnectionParameters) GetSshConfig() *ssh.ClientConfig {
	return s.sshConfig
}

func (s *SshConnectionParameters) GetAddress() string {
	return s.address
}
