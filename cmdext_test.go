package exec_test

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestGoWrapper(t *testing.T) {
	cmd := exec.Command("go", "version")
	expected, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%q failed: %v", cmd, err)
	}

	cmd = exec.Command("go", "run", "./_examples/go-wrapper", "version")
	actual, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%q failed: %v", cmd, err)
	}

	if !bytes.Equal(actual, expected) {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}
