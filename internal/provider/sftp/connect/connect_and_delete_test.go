package connect

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

type mockDeleteInputModel struct {
	path types.String
}

func (m *mockDeleteInputModel) GetPath() types.String {
	return m.path
}

func TestConnectAndDeleteOperation_ExistingFile(t *testing.T) {
	server, serverAddr, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create a test file that we'll then delete
	testFilePath := filepath.Join(server.testDir, "to_delete.txt")
	err := os.WriteFile(testFilePath, []byte("delete me"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file for deletion: %v", err)
	}
	
	// Verify the file exists before deletion
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Fatalf("Test file does not exist before deletion test: %v", err)
	}

	// Set up the delete operation
	input := &mockDeleteInputModel{
		path: types.StringValue("to_delete.txt"),
	}

	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	operation := ConnectAndDelete(
		sshParams,
		input,
	)

	// Execute the delete operation
	err = operation()
	if err != nil {
		t.Errorf("ConnectAndDeleteOperation() error = %v, expected no error", err)
		return
	}

	// Verify the file has been deleted
	if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
		t.Errorf("Expected file to be deleted, but it still exists")
	}
}

func TestConnectAndDeleteOperation_NonExistentFile(t *testing.T) {
	server, serverAddr, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Set up the delete operation for a non-existent file
	input := &mockDeleteInputModel{
		path: types.StringValue("non_existent.txt"),
	}

	sshParams := &mockSSHParams{
		config:  getTestClientConfig(server.hostPrivateKey.PublicKey()),
		address: serverAddr,
	}

	operation := ConnectAndDelete(
		sshParams,
		input,
	)

	// Execute the delete operation
	err := operation()
	if err != nil {
		t.Errorf("ConnectAndDeleteOperation() error = %v, expected no error for non-existent file", err)
	}
}

func TestIsFileNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "file not found error",
			err:      fmt.Errorf("no such file or directory"),
			expected: true,
		},
		{
			name:     "file does not exist error",
			err:      fmt.Errorf("file does not exist"),
			expected: true,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("permission denied"),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsFileNotFound(tc.err)
			if result != tc.expected {
				t.Errorf("IsFileNotFound() = %v, expected %v for error: %v", result, tc.expected, tc.err)
			}
		})
	}
}

func TestConnectAndDeleteOperation_ConnectionFailure(t *testing.T) {
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

	input := &mockDeleteInputModel{
		path: types.StringValue("some_file.txt"),
	}

	operation := ConnectAndDelete(
		sshParams,
		input,
	)

	// Execute the delete operation, expect an error
	err = operation()
	if err == nil {
		t.Errorf("ConnectAndDeleteOperation() expected error for connection failure, got nil")
	}
}
