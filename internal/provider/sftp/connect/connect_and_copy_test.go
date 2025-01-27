package connect

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type testServer struct {
	sshServer      *ssh.ServerConfig
	listener       net.Listener
	testDir        string
	hostPrivateKey ssh.Signer
}

type mockInputModel struct {
	path         types.String
	allowMissing types.Bool
}

func (m *mockInputModel) GetPath() types.String {
	return m.path
}

func (m *mockInputModel) GetAllowMissing() types.Bool {
	return m.allowMissing
}

type mockOutputModel struct {
	id           types.String
	contents     types.String
	lastModified types.String
	size         types.Int64
}

func (m *mockOutputModel) SetID(id types.String) {
	m.id = id
}

func (m *mockOutputModel) GetID() types.String {
	return m.id
}

func (m *mockOutputModel) SetContents(contents types.String) {
	m.contents = contents
}

func (m *mockOutputModel) GetContents() types.String {
	return m.contents
}

func (m *mockOutputModel) SetLastModified(lastModified types.String) {
	m.lastModified = lastModified
}

func (m *mockOutputModel) GetLastModified() types.String {
	return m.lastModified
}

func (m *mockOutputModel) SetSize(size types.Int64) {
	m.size = size
}

func (m *mockOutputModel) GetSize() types.Int64 {
	return m.size
}

// Mock SSH connection parameters
type mockSSHParams struct {
	config  *ssh.ClientConfig
	address string
}

func (m *mockSSHParams) GetSshConfig() *ssh.ClientConfig {
	return m.config
}

func (m *mockSSHParams) GetAddress() string {
	return m.address
}

func setupTestServer(t *testing.T) (*testServer, error) {
	// Create temporary directory for test files
	testDir, err := os.MkdirTemp("", "sftp_test")
	if err != nil {
		return nil, fmt.Errorf("failed to create test directory: %v", err)
	}

	// Generate SSH host key
	rawKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		os.RemoveAll(testDir)
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	hostKey, err := ssh.NewSignerFromKey(rawKey)
	if err != nil {
		os.RemoveAll(testDir)
		return nil, fmt.Errorf("failed to create host key signer: %v", err)
	}

	// Configure SSH server
	sshConfig := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if string(pass) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}
	sshConfig.AddHostKey(hostKey)

	// Start SSH server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.RemoveAll(testDir)
		return nil, fmt.Errorf("failed to listen for connection: %v", err)
	}

	go func() {
		for {
			nConn, err := listener.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					t.Errorf("failed to accept connection: %v", err)
				}
				return
			}

			go handleConnection(t, nConn, sshConfig, testDir)
		}
	}()

	return &testServer{
		sshServer:      sshConfig,
		listener:       listener,
		testDir:        testDir,
		hostPrivateKey: hostKey,
	}, nil
}

func handleConnection(t *testing.T, conn net.Conn, sshConfig *ssh.ServerConfig, rootDir string) {
	defer conn.Close()

	// Handle SSH connection
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshConfig)
	if err != nil {
		t.Errorf("failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	// Discard incoming requests
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			t.Errorf("failed to accept channel: %v", err)
			return
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				ok := false
				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "sftp" {
						ok = true
						go handleSftp(t, channel, rootDir)
					}
				}
				req.Reply(ok, nil)
			}
		}(requests)
	}
}

func handleSftp(t *testing.T, channel ssh.Channel, rootDir string) {
	server, err := sftp.NewServer(
		channel,
		sftp.WithServerWorkingDirectory(rootDir),
	)
	if err != nil {
		t.Errorf("failed to create SFTP server: %v", err)
		return
	}
	defer server.Close()

	if err := server.Serve(); err != nil && err != io.EOF {
		t.Errorf("server exited with error: %v", err)
	}
}

func (ts *testServer) cleanup() {
	if ts.listener != nil {
		ts.listener.Close()
	}
	if ts.testDir != "" {
		os.RemoveAll(ts.testDir)
	}
}

func getTestClientConfig(hostKey ssh.PublicKey) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.Password("testpass"),
		},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}
}

// setupIntegrationTest prepares the test environment and returns cleanup function
func setupIntegrationTest(t *testing.T) (*testServer, string, string, func()) {
	t.Helper()

	server, err := setupTestServer(t)
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	serverAddr := server.listener.Addr().String()

	// Create test file without leading slash - this will be relative to testDir
	testContent := "test content\n"
	testFilePath := filepath.Join(server.testDir, "test.txt")

	t.Logf("Creating test file at: %s", testFilePath)
	err = os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cleanup := func() {
		t.Logf("Cleaning up test server and directory: %s", server.testDir)
		server.cleanup()
	}

	return server, serverAddr, testContent, cleanup
}

func TestConnectAndCopyOperation_ExistingFile(t *testing.T) {
	server, serverAddr, testContent, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Use "test.txt" without leading slash - the path should be relative to SFTP root
	input := &mockInputModel{
		path:         types.StringValue("test.txt"),
		allowMissing: types.BoolValue(false),
	}

	output := &mockOutputModel{}
	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	t.Logf("Test server directory: %s", server.testDir)
	t.Logf("Attempting to access file: %s", input.path.ValueString())

	operation := ConnectAndCopy(
		sshParams,
		input,
		output,
	)

	err := operation()
	if err != nil {
		t.Errorf("ConnectAndCopyOperation() error = %v, expected no error", err)
		return
	}

	// Verify output
	if output.GetID().ValueString() != "test.txt" {
		t.Errorf("expected ID 'test.txt', got %s", output.GetID().ValueString())
	}
	if output.GetContents().ValueString() != testContent {
		t.Errorf("expected content %q, got %q", testContent, output.GetContents().ValueString())
	}
	if output.GetSize().ValueInt64() != int64(len(testContent)) {
		t.Errorf("expected size %d, got %d", len(testContent), output.GetSize().ValueInt64())
	}
}

func TestConnectAndCopyOperation_MissingFileAllowed(t *testing.T) {
	server, serverAddr, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	input := &mockInputModel{
		path:         types.StringValue("missing.txt"),
		allowMissing: types.BoolValue(true),
	}
	output := &mockOutputModel{}
	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	operation := ConnectAndCopy(
		sshParams,
		input,
		output,
	)

	err := operation()
	if err != nil {
		t.Errorf("ConnectAndCopyOperation() error = %v, expected no error", err)
		return
	}

	// Verify output
	if output.GetID().ValueString() != "missing" {
		t.Errorf("expected ID 'missing', got %s", output.GetID().ValueString())
	}
	if output.GetContents().ValueString() != "" {
		t.Errorf("expected empty content, got %q", output.GetContents().ValueString())
	}
	if output.GetSize().ValueInt64() != -1 {
		t.Errorf("expected size -1, got %d", output.GetSize().ValueInt64())
	}
}

func TestConnectAndCopyOperation_MissingFileNotAllowed(t *testing.T) {
	server, serverAddr, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	input := &mockInputModel{
		path:         types.StringValue("missing.txt"),
		allowMissing: types.BoolValue(false),
	}
	output := &mockOutputModel{}
	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	operation := ConnectAndCopy(
		sshParams,
		input,
		output,
	)

	err := operation()
	if err == nil {
		t.Error("ConnectAndCopyOperation() expected error for missing file, got nil")
	}
}
