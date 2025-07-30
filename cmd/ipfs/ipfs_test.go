package ipfs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/spf13/cobra"
)

// Helper function to get a pinned CID for testing
func getTestPinnedCID(t *testing.T) string {
	cid, err := GetPinnedCID()
	if err != nil {
		t.Skipf("⚠️ Skipping test - no pinned CIDs available: %v", err)
	}
	t.Logf("🔍 Using pinned CID: %s", cid)
	return cid
}

// Helper function to get an unpinned CID for testing
func getTestUnpinnedCID(t *testing.T) string {
	cid, err := GetUnpinnedCID()
	if err != nil {
		t.Skipf("⚠️ Skipping test - no unpinned CIDs available: %v", err)
	}
	t.Logf("🔍 Using unpinned CID: %s", cid)
	return cid
}

// extractCIDFromOutput extracts the CID from the load-file command output
func extractCIDFromOutput(output string) (string, bool) {
	if !strings.Contains(output, "📁 IPFS Path: /ipfs/") {
		return "", false
	}
	
	// Extract CID from the output
	startIndex := strings.Index(output, "/ipfs/")
	if startIndex == -1 {
		return "", false
	}
	
	cidPath := output[startIndex:] // Get the full path starting with "/ipfs/"
	// Find the end of the CID (newline or space)
	endIndex := strings.IndexAny(cidPath, "\n\r\t ")
	if endIndex != -1 {
		cidPath = cidPath[:endIndex]
	}
	
	return cidPath, true
}



func exec(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	_, err := cmd.ExecuteC()
	return buf.String(), err
}

func TestNewCmd(t *testing.T) {
	
	cmd := NewCmd()
	
	if cmd.Use != "ipfs" {
		t.Errorf("Expected command use to be 'ipfs', got '%s'", cmd.Use)
	}
	
	if !strings.Contains(cmd.Short, "IPFS") {
		t.Errorf("Expected command short description to contain 'IPFS', got '%s'", cmd.Short)
	}
	
	t.Log("✅ Command structure validation passed")
	
	// Check that subcommands are added
	subcommands := cmd.Commands()
	if len(subcommands) != 3 {
		t.Errorf("Expected 3 subcommands, got %d", len(subcommands))
	}
	
	// Check for specific subcommands
	foundCheckPins := false
	foundLoadFile := false
	foundCheckCid := false
	for _, sub := range subcommands {
		switch sub.Use {
		case "check-pins":
			foundCheckPins = true
			t.Log("✅ Found check-pins subcommand")
		case "load-file <path>":
			foundLoadFile = true
			t.Log("✅ Found load-file subcommand")
		case "check-cid <cid>":
			foundCheckCid = true
			t.Log("✅ Found check-cid subcommand")
		}
	}
	
	if !foundCheckPins {
		t.Error("check-pins subcommand not found")
	}
	if !foundLoadFile {
		t.Error("load-file subcommand not found")
	}
	if !foundCheckCid {
		t.Error("check-cid subcommand not found")
	}
}

func TestCheckPinsCmd(t *testing.T) {
	
	root := NewCmd()
	
	// Test basic command execution
	t.Log("📋 Executing check-pins command...")
	out, err := exec(root, "check-pins")
	if err != nil {
		t.Errorf("Expected no error returned, but got: %v", err)
	}
	
	// Check that the command starts with the expected message
	if !strings.Contains(out, "--- Pin Check Summary ---") {
		t.Errorf("Expected output to contain '--- Pin Check Summary ---', got: %q", out)
	}

	t.Log("✅ check-pins command executed successfully")
}

func TestCheckCidCmd(t *testing.T) {
	
	root := NewCmd()
	
	// Test with missing argument
	t.Log("📋 Testing check-cid with missing argument...")
	_, err := exec(root, "check-cid")
	if err == nil {
		t.Error("Expected error when no CID provided")
	} else {
		t.Log("✅ Correctly received error for missing argument")
	}

	
	// Test with invalid CID format
	t.Log("📋 Testing check-cid with invalid CID format...")
	_, err = exec(root, "check-cid", "invalid-cid")
	if err != nil {
		if !strings.Contains(err.Error(), "invalid path") {
			t.Errorf("Expected 'Error: invalid path' error, but got: %v", err)
		} else {
			t.Log("✅ Correctly received error for invalid CID format")
		}
	}

	// Test with unpinned valid CID format
	t.Log("📋 Testing check-cid with unpinned valid CID...")
	unpinnedCID := getTestUnpinnedCID(t)
	
	out, err := exec(root, "check-cid", unpinnedCID)
	if err != nil {
		t.Errorf("No error expected, got: %v", err)
	}
	// Check that the command starts with the expected message
	if !strings.Contains(out, "CID is NOT pinned") {
		t.Errorf("Expected not pinned cid, got: %q", out)
	} else {
		t.Log("✅ Correctly identified unpinned CID")
	}
}

func TestCheckCidCmdWithPinnedCID(t *testing.T) {
	
	root := NewCmd()
	
	// Test with pinned valid CID format
	pinnedCID := getTestPinnedCID(t)
	
	out, err := exec(root, "check-cid", pinnedCID)
	if err != nil {
		t.Errorf("No error expected, got: %v", err)
	}

	// Check that the command starts with the expected message
	if !strings.Contains(out, "CID is pinned") {
		t.Errorf("Expected pinned cid, got: %q", out)
	} else {
		t.Log("✅ Correctly identified pinned CID")
	}
}

func TestLoadFileCmd(t *testing.T) {
	
	root := NewCmd()
	
	// Test with missing argument
	t.Log("📋 Testing load-file with missing argument...")
	_, err := exec(root, "load-file")
	if err == nil {
		t.Error("Expected error when no file path provided")
	} else {
		t.Log("✅ Correctly received error for missing argument")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Errorf("Expected argument error, got: %v", err)
	}
	
	// Test with non-existent file
	t.Log("📋 Testing load-file with non-existent file...")
	_, err = exec(root, "load-file", "non-existent-file.txt")
	if err == nil {
		t.Error("Expected error when file doesn't exist")
	} else {
		t.Log("✅ Correctly received error for non-existent file")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Logf("Expected file read error, but got: %v", err)
	}
}

func TestLoadFileCmdWithValidFile(t *testing.T) {
	
	// Create a test file with known content
	testContent := "Test content for IPFS upload"
	
	tmpFile, err := os.CreateTemp("", "ipfs-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	t.Logf("📁 Created temporary file: %s", tmpFile.Name())
	
	_, err = tmpFile.WriteString(testContent)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()
	
	t.Log("📤 Uploading file to IPFS...")
	root := NewCmd()
	out, err := exec(root, "load-file", tmpFile.Name())
	if err != nil {
		t.Errorf("No error expected, got: %v", err)
	}

	// The command should start successfully and show file size
	if !strings.Contains(out, "File size:") {
		t.Errorf("Expected output to contain file size information, got: %q", out)
	} else {
		t.Log("✅ File size information found in output")
	}
	
	// Should show the file size of our test content
	expectedSize := len(testContent)
	if !strings.Contains(out, fmt.Sprintf("%d bytes", expectedSize)) {
		t.Errorf("Expected output to contain file size %d, got: %q", expectedSize, out)
	} else {
		t.Logf("✅ Correct file size (%d bytes) found in output", expectedSize)
	}

	// Remove the file from IPFS
	if cidPath, found := extractCIDFromOutput(out); found {
		t.Logf("🗑️  Unpinning %v file from IPFS...", cidPath)
		
		// Unpin the CID using IPFS API
		api, err := rpc.NewLocalApi()
		if err != nil {
			t.Logf("❌ Failed to connect to IPFS node for unpinning: %v", err)
		} else {
			ctx := context.Background()
			ipfsPath, err := path.NewPath(cidPath)
			if err != nil {
				t.Logf("❌ Failed to create CID path: %v", err)
			} else {
				err = api.Pin().Rm(ctx, ipfsPath)
				if err != nil {
					t.Logf("❌ Failed to unpin CID path: %s: %v", cidPath, err)
				} else {
					t.Logf("✅ Successfully unpinned CID path: %s", cidPath)
				}
			}
		}
	} else {
		t.Log("⚠️  No IPFS path found in output - skipping cleanup")
	}
}

func TestCommandHelp(t *testing.T) {
	
	root := NewCmd()
	
	// Test help for main command
	out, err := exec(root, "--help")
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}
	if !strings.Contains(out, "IPFS-related utilities") {
		t.Errorf("Help should contain command description, got: %q", out)
	} else {
		t.Log("✅ Main command help contains expected description")
	}
	
	// Test help for check-pins subcommand
	out, err = exec(root, "check-pins", "--help")
	if err != nil {
		t.Fatalf("check-pins help failed: %v", err)
	}
	if !strings.Contains(out, "Verify that pins are still intact") {
		t.Errorf("check-pins help should contain description, got: %q", out)
	} else {
		t.Log("✅ check-pins help contains expected description")
	}
	
	// Test help for load-file subcommand
	out, err = exec(root, "load-file", "--help")
	if err != nil {
		t.Fatalf("load-file help failed: %v", err)
	}
	if !strings.Contains(out, "Add a local file to IPFS and pin it") {
		t.Errorf("load-file help should contain description, got: %q", out)
	} else {
		t.Log("✅ load-file help contains expected description")
	}
	
	// Test help for check-cid subcommand
	out, err = exec(root, "check-cid", "--help")
	if err != nil {
		t.Fatalf("check-cid help failed: %v", err)
	}
	if !strings.Contains(out, "Check if a specific CID exists in IPFS") {
		t.Errorf("check-cid help should contain description, got: %q", out)
	} else {
		t.Log("✅ check-cid help contains expected description")
	}
}

func TestUtilsFunctions(t *testing.T) {
	
	// Test GetPinnedCID function
	t.Log("📋 Testing GetPinnedCID function...")
	cid, err := GetPinnedCID()
	if err != nil {
		if(strings.Contains(err.Error(), "no pinned CIDs found")){
			t.Logf("⚠️  No pinned CIDs available: %v", err)
		} else {
			t.Errorf("❌ Error getting pinned CID: %v", err)
		}
	} else {
		t.Logf("🔍 Found pinned CID: %s", cid)
	
		// Verify the returned CID is valid
		if err := ValidateCID(cid); err != nil {
			t.Errorf("❌ Invalid CID: %v", err)
		} else {
			t.Log("✅ Pinned CID validation passed")
		}
	}
	


	// test GetUnpinnedCID function
	t.Log("📋 Testing GetUnpinnedCID function...")
	unpinnedCID, err := GetUnpinnedCID()
	if err != nil {
		t.Logf("⚠️  No unpinned CIDs available: %v", err)
		return
	}

	t.Logf("🔍 Found unpinned CID: %s", unpinnedCID)
	
	// Verify the returned CID is valid
	if err := ValidateCID(unpinnedCID); err != nil {
		t.Errorf("❌ Invalid CID: %v", err)
	} else {
		t.Log("✅ Unpinned CID validation passed")
	}
}
