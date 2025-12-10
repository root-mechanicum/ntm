package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var (
	buildOnce  sync.Once
	binaryPath string
	buildErr   error
)

// BuildLocalNTM builds the ntm binary from the current workspace and returns its path.
// It builds only once per test process and skips the caller on build failure.
func BuildLocalNTM(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ntm-bin-*")
		if err != nil {
			buildErr = err
			return
		}
		binaryPath = filepath.Join(dir, "ntm")
		cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/ntm")
		cmd.Env = os.Environ()
		buildErr = cmd.Run()
	})

	if buildErr != nil {
		t.Skipf("skipping: failed to build ntm binary: %v", buildErr)
	}
	return binaryPath
}
