package app

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

func TestStartCmd(t *testing.T) {
	root := NewCmd() // app sub-tree
	out, err := exec(root, "start")
	if err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	if !strings.Contains(out, "DCS app starting") {
		t.Errorf("got %q, want substring %q", out, "DCS app starting")
	}
}
