package connect

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

type mockWriteInputModel struct {
	path        types.String
	contents    types.String
	permissions types.String
}

func (m *mockWriteInputModel) GetPath() types.String {
	return m.path
}

func (m *mockWriteInputModel) GetContents() types.String {
	return m.contents
}

func (m *mockWriteInputModel) GetPermissions() types.String {
	return m.permissions
}

func TestConnectAndWriteOperation_NewFile(t *testing.T) {
	server, serverAddr, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	testContent := "new file content"
	testFilename := "new_write_test.txt"
	testFilePath := filepath.Join(server.testDir, testFilename)

	// Set up the write operation
	input := &mockWriteInputModel{
		path:        types.StringValue(testFilename),
		contents:    types.StringValue(testContent),
		permissions: types.StringValue("0644"),
	}

	output := &mockOutputModel{}
	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	operation := ConnectAndWrite(
		sshParams,
		input,
		output,
	)

	// Execute the write operation
	err := operation()
	if err != nil {
		t.Errorf("ConnectAndWriteOperation() error = %v, expected no error", err)
		return
	}

	// Verify file was created with correct content
	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Errorf("Failed to read created file: %v", err)
		return
	}

	if string(content) != testContent {
		t.Errorf("File content = %q, expected %q", string(content), testContent)
	}

	// Verify output model was updated correctly
	if output.GetID().ValueString() != testFilename {
		t.Errorf("output.ID = %q, expected %q", output.GetID().ValueString(), testFilename)
	}

	if output.GetContents().ValueString() != testContent {
		t.Errorf("output.Contents = %q, expected %q", output.GetContents().ValueString(), testContent)
	}

	if output.GetSize().ValueInt64() != int64(len(testContent)) {
		t.Errorf("output.Size = %d, expected %d", output.GetSize().ValueInt64(), len(testContent))
	}

	// Check file permissions
	fileInfo, err := os.Stat(testFilePath)
	if err != nil {
		t.Errorf("Failed to get file info: %v", err)
		return
	}

	// On Unix/Linux, we can check if permissions match expected value (0644 = rw-r--r--)
	if fileInfo.Mode().Perm() != 0644 {
		t.Errorf("File permissions = %o, expected %o", fileInfo.Mode().Perm(), 0644)
	}
}

func TestConnectAndWriteOperation_OverwriteExistingFile(t *testing.T) {
	server, serverAddr, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create initial file with some content
	testFilename := "overwrite_test.txt"
	testFilePath := filepath.Join(server.testDir, testFilename)
	initialContent := "initial content"
	err := os.WriteFile(testFilePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial test file: %v", err)
	}

	// Prepare new content to overwrite
	newContent := "new overwritten content"

	// Set up the write operation
	input := &mockWriteInputModel{
		path:        types.StringValue(testFilename),
		contents:    types.StringValue(newContent),
		permissions: types.StringValue("0600"),
	}

	output := &mockOutputModel{}
	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	operation := ConnectAndWrite(
		sshParams,
		input,
		output,
	)

	// Execute the write operation
	err = operation()
	if err != nil {
		t.Errorf("ConnectAndWriteOperation() error = %v, expected no error", err)
		return
	}

	// Verify file was overwritten with new content
	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Errorf("Failed to read overwritten file: %v", err)
		return
	}

	if string(content) != newContent {
		t.Errorf("File content = %q, expected %q", string(content), newContent)
	}

	// Verify output model was updated correctly
	if output.GetID().ValueString() != testFilename {
		t.Errorf("output.ID = %q, expected %q", output.GetID().ValueString(), testFilename)
	}

	if output.GetContents().ValueString() != newContent {
		t.Errorf("output.Contents = %q, expected %q", output.GetContents().ValueString(), newContent)
	}

	// Check file permissions were changed
	fileInfo, err := os.Stat(testFilePath)
	if err != nil {
		t.Errorf("Failed to get file info: %v", err)
		return
	}

	// On Unix/Linux, we can check if permissions match expected value (0600 = rw-------)
	if fileInfo.Mode().Perm() != 0600 {
		t.Errorf("File permissions = %o, expected %o", fileInfo.Mode().Perm(), 0600)
	}
}

func TestConnectAndWriteOperation_WithoutPermissions(t *testing.T) {
	server, serverAddr, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	testContent := "content without explicit permissions"
	testFilename := "no_perm_test.txt"

	// Set up the write operation without specifying permissions
	input := &mockWriteInputModel{
		path:        types.StringValue(testFilename),
		contents:    types.StringValue(testContent),
		permissions: types.StringNull(),
	}

	output := &mockOutputModel{}
	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	operation := ConnectAndWrite(
		sshParams,
		input,
		output,
	)

	// Execute the write operation
	err := operation()
	if err != nil {
		t.Errorf("ConnectAndWriteOperation() error = %v, expected no error", err)
		return
	}

	// Just verify file was created with correct content
	testFilePath := filepath.Join(server.testDir, testFilename)
	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Errorf("Failed to read created file: %v", err)
		return
	}

	if string(content) != testContent {
		t.Errorf("File content = %q, expected %q", string(content), testContent)
	}
}

func TestConnectAndWriteOperation_InvalidPermissions(t *testing.T) {
	server, serverAddr, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Set up the write operation with invalid permissions
	input := &mockWriteInputModel{
		path:        types.StringValue("invalid_perm.txt"),
		contents:    types.StringValue("test content"),
		permissions: types.StringValue("invalid"),
	}

	output := &mockOutputModel{}
	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	operation := ConnectAndWrite(
		sshParams,
		input,
		output,
	)

	// Execute the write operation, expect error due to invalid permissions
	err := operation()
	if err == nil {
		t.Errorf("ConnectAndWriteOperation() expected error for invalid permissions, got nil")
	}
}

func TestConnectAndWriteOperation_ConnectionFailure(t *testing.T) {
	// Create a mock SSH connection parameters with an invalid address
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	sshParams := &mockSSHParams{
		config: &ssh.ClientConfig{
			User: "testuser",
			Auth: []ssh.AuthMethod{
				ssh.Password("testpass"),
			},
			HostKeyCallback: ssh.FixedHostKey(signer.PublicKey()),
		},
		address: "127.0.0.1:1", // Invalid port
	}

	input := &mockWriteInputModel{
		path:     types.StringValue("some_file.txt"),
		contents: types.StringValue("content"),
	}

	output := &mockOutputModel{}

	operation := ConnectAndWrite(
		sshParams,
		input,
		output,
	)

	// Execute the write operation, expect error
	err = operation()
	if err == nil {
		t.Errorf("ConnectAndWriteOperation() expected error for connection failure, got nil")
	}
}
