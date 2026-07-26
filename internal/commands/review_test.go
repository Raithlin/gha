package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// getBinaryPath returns the path to the gha binary
func getBinaryPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return "./bin/gha" // fallback
	}
	// When running `go test ./internal/commands`, the current directory is the package directory.
	// We need to go up two levels to reach the project root, then into bin/gha.
	return filepath.Join(dir, "..", "..", "bin", "gha")
}

// Test the review command with PR number argument
func TestReviewCommand_WithPRNumber(t *testing.T) {
	// Get the binary path
	binaryPath := getBinaryPath()
	
	// Run the command and capture output
	cmd := exec.Command(binaryPath, "review", "123")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute %s: %v", binaryPath, err)
	}
	
	outputStr := string(output)
	assert.Contains(t, outputStr, "Reviewing PR #123")
	assert.Contains(t, outputStr, "⚠️  Implementation pending - this is a placeholder")
}

// Test the review command with assigned flag
func TestReviewCommand_WithAssignedFlag(t *testing.T) {
	// Get the binary path
	binaryPath := getBinaryPath()
	
	// Run the command and capture output
	cmd := exec.Command(binaryPath, "review", "--assigned")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute %s: %v", binaryPath, err)
	}
	
	outputStr := string(output)
	assert.Contains(t, outputStr, "PRs assigned to you:")
	assert.Contains(t, outputStr, "⚠️  Implementation pending - this is a placeholder")
}

// Test the review command with queue flag
func TestReviewCommand_WithQueueFlag(t *testing.T) {
	// Get the binary path
	binaryPath := getBinaryPath()
	
	// Run the command and capture output
	cmd := exec.Command(binaryPath, "review", "--queue")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute %s: %v", binaryPath, err)
	}
	
	outputStr := string(output)
	assert.Contains(t, outputStr, "Review Queue:")
	assert.Contains(t, outputStr, "⚠️  Implementation pending - this is a placeholder")
}

// Test the review command with mine flag
func TestReviewCommand_WithMineFlag(t *testing.T) {
	// Get the binary path
	binaryPath := getBinaryPath()
	
	// Run the command and capture output
	cmd := exec.Command(binaryPath, "review", "--mine")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute %s: %v", binaryPath, err)
	}
	
	outputStr := string(output)
	assert.Contains(t, outputStr, "Your PRs:")
	assert.Contains(t, outputStr, "⚠️  Implementation pending - this is a placeholder")
}

// Test the review command help
func TestReviewCommand_Help(t *testing.T) {
	// Get the binary path
	binaryPath := getBinaryPath()
	
	// Run the command and capture output
	cmd := exec.Command(binaryPath, "review", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute %s: %v", binaryPath, err)
	}
	
	outputStr := string(output)
	assert.Contains(t, outputStr, "Assist with reviewing pull requests")
	assert.Contains(t, outputStr, "Usage:")
	assert.Contains(t, outputStr, "Flags:")
	assert.Contains(t, outputStr, "-a, --assigned")
	assert.Contains(t, outputStr, "-q, --queue")
	assert.Contains(t, outputStr, "-m, --mine")
}