package ipfs

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func exec(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	_, err := cmd.ExecuteC()
	return buf.String(), err
}

func TestCheckPinsCmd(t *testing.T) {
	root := NewCmd()
	out, err := exec(root, "check-pins")
	if err != nil {
		t.Fatalf("check-pins error: %v", err)
	}
	if !strings.Contains(out, "Checking IPFS pins") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestLoadFileCmd(t *testing.T) {
	root := NewCmd()
	out, err := exec(root, "load-file", "dummy.txt")
	if err != nil {
		t.Fatalf("load-file error: %v", err)
	}
	if !strings.Contains(out, "Loading dummy.txt into IPFS") {
		t.Errorf("unexpected output: %q", out)
	}
}
