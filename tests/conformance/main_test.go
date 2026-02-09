package conformance_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var serverURL string

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	tmpDir, err := os.MkdirTemp("", "notspot-conformance-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create tmpdir: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	binPath := filepath.Join(tmpDir, "notspot")

	// Build the binary from source.
	build := exec.Command("go", "build", "-o", binPath, "./cmd/hubspot")
	build.Dir = findModuleRoot()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build binary: %v\n", err)
		return 1
	}

	// Pick a random free port.
	port, err := freePort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find free port: %v\n", err)
		return 1
	}

	addr := fmt.Sprintf(":%d", port)
	serverURL = fmt.Sprintf("http://localhost:%d", port)

	// Start the server with in-memory SQLite.
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"NOTSPOT_ADDR="+addr,
		"NOTSPOT_DB=:memory:",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start server: %v\n", err)
		return 1
	}

	// Wait for server to be ready.
	if err := waitForServer(serverURL, 5*time.Second); err != nil {
		_ = cmd.Process.Kill()
		fmt.Fprintf(os.Stderr, "server not ready: %v\n", err)
		return 1
	}

	code := m.Run()

	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	return code
}

// freePort returns a random available TCP port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	tcpAddr, ok := l.Addr().(*net.TCPAddr)
	_ = l.Close()
	if !ok {
		return 0, fmt.Errorf("expected *net.TCPAddr, got %T", l.Addr())
	}
	return tcpAddr.Port, nil
}

// waitForServer polls the server until it responds or the timeout is reached.
func waitForServer(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/_notspot/reset")
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server at %s did not become ready within %s", baseURL, timeout)
}

// findModuleRoot walks up from the current directory to find go.mod.
func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
