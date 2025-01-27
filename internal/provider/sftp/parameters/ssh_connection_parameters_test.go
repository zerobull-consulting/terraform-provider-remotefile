package parameters

import (
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

type parametersSubset struct {
	Host       types.String
	HostKey    types.String
	Password   types.String
	PrivateKey types.String
	Timeout    types.String
	Port       types.Int64
	User       types.String
}

func (p *parametersSubset) GetHost() types.String {
	return p.Host
}
func (p *parametersSubset) GetHostKey() types.String {
	return p.HostKey
}
func (p *parametersSubset) GetPassword() types.String {
	return p.Password
}
func (p *parametersSubset) GetPrivateKey() types.String {
	return p.PrivateKey
}
func (p *parametersSubset) GetTimeout() types.String {
	return p.Timeout
}
func (p *parametersSubset) GetPort() types.Int64 {
	return p.Port
}
func (p *parametersSubset) GetUser() types.String {
	return p.User
}

func TestMissingPasswordAndPrivateKey(t *testing.T) {
	data := &parametersSubset{
		Host:       types.StringNull(),
		HostKey:    types.StringNull(),
		Password:   types.StringNull(),
		PrivateKey: types.StringNull(),
		Timeout:    types.StringNull(),
		Port:       types.Int64Null(),
		User:       types.StringNull(),
	}
	_, err := CreateSSHConnectionParameters(data)
	if err == nil {
		t.Error("expected an error when both password and private key are missing")
	}
	if err.Error() != "must provide either a password or private key" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvalidPrivateKey(t *testing.T) {
	data := &parametersSubset{
		Host:       types.StringNull(),
		HostKey:    types.StringNull(),
		Password:   types.StringNull(),
		PrivateKey: types.StringValue("invalid"),
		Timeout:    types.StringNull(),
		Port:       types.Int64Null(),
		User:       types.StringNull(),
	}
	_, err := CreateSSHConnectionParameters(data)
	if err == nil {
		t.Error("expected an error when the private key is supplied but invalid")
	}
	if err.Error() != "failed to parse private key: ssh: no key found" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvalidHostKey(t *testing.T) {
	data := &parametersSubset{
		Host:       types.StringNull(),
		HostKey:    types.StringValue("invalid"),
		Password:   types.StringValue("password"),
		PrivateKey: types.StringNull(),
		Timeout:    types.StringNull(),
		Port:       types.Int64Null(),
		User:       types.StringNull(),
	}
	_, err := CreateSSHConnectionParameters(data)
	if err == nil {
		t.Error("expected an error when the host key is invalid")
	}
	if !strings.Contains(err.Error(), "failed to parse host key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvalidTimeout(t *testing.T) {
	data := &parametersSubset{
		Host:       types.StringNull(),
		HostKey:    types.StringNull(),
		Password:   types.StringValue("password"),
		PrivateKey: types.StringNull(),
		Timeout:    types.StringValue("invalid"),
		Port:       types.Int64Null(),
		User:       types.StringNull(),
	}
	_, err := CreateSSHConnectionParameters(data)
	if err == nil {
		t.Error("expected an error when the timeout is invalid")
	}
	if !strings.Contains(err.Error(), "invalid timeout duration") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPasswordConfigCreated(t *testing.T) {
	data := &parametersSubset{
		Host:       types.StringValue("host"),
		HostKey:    types.StringNull(),
		Password:   types.StringValue("password"),
		PrivateKey: types.StringNull(),
		Timeout:    types.StringNull(),
		Port:       types.Int64Null(),
		User:       types.StringValue("user"),
	}
	params, err := CreateSSHConnectionParameters(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if params == nil {
		t.Error("expected connection parameters to be created")
		return
	}
	if params.address != "host:22" {
		t.Errorf("unexpected address: %s", params.address)
	}
	if params.sshConfig.User != "user" {
		t.Errorf("unexpected user: %s", params.sshConfig.User)
	}
	if len(params.sshConfig.Auth) != 1 {
		t.Errorf("unexpected number of auth methods: %d", len(params.sshConfig.Auth))
	}
}

func TestPrivateKeyConfigCreated(t *testing.T) {
	pemKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACDcOHrH1gUk6C6gqdp9f4TqWirZJ06HuulVC9emObD1sgAAALBeRtf/XkbX
/wAAAAtzc2gtZWQyNTUxOQAAACDcOHrH1gUk6C6gqdp9f4TqWirZJ06HuulVC9emObD1sg
AAAEBmL3FBBSsrsIG13mHt26KsddW2ARnLu+7mLcLagsL5Rtw4esfWBSToLqCp2n1/hOpa
KtknToe66VUL16Y5sPWyAAAAKWRldmVsb3BlckB1cmJhbi1yZWNydWl0ZXItd2ViLWRldm
Vsb3BtZW50AQIDBA==
-----END OPENSSH PRIVATE KEY-----`

	data := &parametersSubset{
		Host:       types.StringValue("host"),
		HostKey:    types.StringNull(),
		Password:   types.StringNull(),
		PrivateKey: types.StringValue(pemKey),
		Timeout:    types.StringNull(),
		Port:       types.Int64Null(),
		User:       types.StringValue("user"),
	}
	params, err := CreateSSHConnectionParameters(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if params == nil {
		t.Error("expected connection parameters to be created")
		return
	}
	if params.address != "host:22" {
		t.Errorf("unexpected address: %s", params.address)
	}
	if params.sshConfig.User != "user" {
		t.Errorf("unexpected user: %s", params.sshConfig.User)
	}
	if len(params.sshConfig.Auth) != 1 {
		t.Errorf("unexpected number of auth methods: %d", len(params.sshConfig.Auth))
	}
}

func TestTimeoutSupplied(t *testing.T) {
	data := &parametersSubset{
		Host:       types.StringValue("host"),
		HostKey:    types.StringNull(),
		Password:   types.StringValue("password"),
		PrivateKey: types.StringNull(),
		Timeout:    types.StringValue("2m"),
		Port:       types.Int64Null(),
		User:       types.StringValue("user"),
	}
	params, err := CreateSSHConnectionParameters(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if params == nil {
		t.Error("expected connection parameters to be created")
		return
	}
	if params.sshConfig.Timeout != time.Minute*2 {
		t.Errorf("unexpected timeout: %v", params.sshConfig.Timeout)
	}
}

func TestPortSupplied(t *testing.T) {
	data := &parametersSubset{
		Host:       types.StringValue("host"),
		HostKey:    types.StringNull(),
		Password:   types.StringValue("password"),
		PrivateKey: types.StringNull(),
		Timeout:    types.StringNull(),
		Port:       types.Int64Value(2222),
		User:       types.StringValue("user"),
	}
	params, err := CreateSSHConnectionParameters(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if params == nil {
		t.Error("expected connection parameters to be created")
		return
	}
	if params.address != "host:2222" {
		t.Errorf("unexpected address: %s", params.address)
	}
}

func TestHostKeySupplied(t *testing.T) {
	key := `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICbM5h6z001fq+dToAcn+jwfXrk+xCHgyiaUc7LJCe4a devel@ubuntu-server`
	newAuthorizedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
	if err != nil {
		t.Errorf("unexpected error in test code, check supplied test host key: %v", err)
	}
	data := &parametersSubset{
		Host:       types.StringValue("host"),
		HostKey:    types.StringValue(string(newAuthorizedKey.Marshal())),
		Password:   types.StringValue("password"),
		PrivateKey: types.StringNull(),
		Timeout:    types.StringNull(),
		Port:       types.Int64Null(),
		User:       types.StringValue("user"),
	}
	params, err := CreateSSHConnectionParameters(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if params == nil {
		t.Error("expected connection parameters to be created")
		return
	}
	if params.sshConfig.HostKeyCallback == nil {
		t.Error("expected host key callback to be set")
	}
}

func TestAllParamsSupplied(t *testing.T) {
	privateKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACDcOHrH1gUk6C6gqdp9f4TqWirZJ06HuulVC9emObD1sgAAALBeRtf/XkbX
/wAAAAtzc2gtZWQyNTUxOQAAACDcOHrH1gUk6C6gqdp9f4TqWirZJ06HuulVC9emObD1sg
AAAEBmL3FBBSsrsIG13mHt26KsddW2ARnLu+7mLcLagsL5Rtw4esfWBSToLqCp2n1/hOpa
KtknToe66VUL16Y5sPWyAAAAKWRldmVsb3BlckB1cmJhbi1yZWNydWl0ZXItd2ViLWRldm
Vsb3BtZW50AQIDBA==
-----END OPENSSH PRIVATE KEY-----`
	hostKeyPlain := `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICbM5h6z001fq+dToAcn+jwfXrk+xCHgyiaUc7LJCe4a devel@ubuntu-server`
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(hostKeyPlain))
	if err != nil {
		t.Errorf("unexpected error in test code, check supplied test host key: %v", err)
	}
	data := &parametersSubset{
		Host:       types.StringValue("host"),
		HostKey:    types.StringValue(string(hostKey.Marshal())),
		Password:   types.StringValue("password"),
		PrivateKey: types.StringValue(privateKey),
		Timeout:    types.StringValue("2m"),
		Port:       types.Int64Value(2222),
		User:       types.StringValue("ubuntu"),
	}
	params, err := CreateSSHConnectionParameters(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if params == nil {
		t.Error("expected connection parameters to be created")
		return
	}
	if params.address != "host:2222" {
		t.Errorf("unexpected address: %s", params.address)
	}
	if params.sshConfig.User != "ubuntu" {
		t.Errorf("unexpected user: %s", params.sshConfig.User)
	}
	if len(params.sshConfig.Auth) != 1 {
		t.Errorf("unexpected number of auth methods: %d", len(params.sshConfig.Auth))
	}
	if params.sshConfig.Timeout != time.Minute*2 {
		t.Errorf("unexpected timeout: %v", params.sshConfig.Timeout)
	}
	if params.sshConfig.HostKeyCallback == nil {
		t.Error("expected host key callback to be set")
	}
}
