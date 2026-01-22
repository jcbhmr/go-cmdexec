//go:build unix

package exec_test

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestExamplesGoWrapper(t *testing.T) {
	cmd := exec.Command("go", "version")
	expected, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("go", "run", "./_examples/go-wrapper", "version")
	actual, err := cmd.CombinedOutput()
	if !bytes.Equal(expected, actual) {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}
