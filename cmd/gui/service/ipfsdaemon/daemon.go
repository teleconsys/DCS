// Package ipfsdaemon runs and stops a local `ipfs daemon` child process for the GUI.
package ipfsdaemon

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Default is the app-wide IPFS daemon manager.
var Default = New()

// Manager owns at most one ipfs daemon child process started from the GUI.
type Manager struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	managed bool
	onExit  func(err error)
}

// New returns a fresh manager (no process running).
func New() *Manager { return &Manager{} }

// Managed reports whether the daemon was started by this manager and is still running.
func (m *Manager) Managed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.managed
}

// SetOnExit configures a hook invoked when the managed daemon process exits.
func (m *Manager) SetOnExit(fn func(err error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onExit = fn
}

// Reachable reports whether a local IPFS HTTP API responds (any daemon, not only ours).
func Reachable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:5001/api/v0/version", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Start launches `ipfs daemon`. Returns an error if we already manage a process,
// the binary is missing, or the process cannot be started.
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.managed {
		m.mu.Unlock()
		return fmt.Errorf("IPFS daemon is already running under DCS")
	}
	m.mu.Unlock()

	bin, err := resolveBinary()
	if err != nil {
		return err
	}

	cmd := exec.Command(bin, "daemon")
	applyPlatformAttrs(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ipfs daemon: %w", err)
	}

	m.mu.Lock()
	m.cmd = cmd
	m.managed = true
	m.mu.Unlock()

	go discard(stdout)
	go discard(stderr)
	go m.wait(cmd)
	return nil
}

// Shutdown stops a managed daemon if one is running. Intended for app exit;
// errors are ignored and exit callbacks are cleared.
func (m *Manager) Shutdown() {
	m.SetOnExit(nil)
	_ = m.Stop()
}

// Shutdown stops Default if this app started the daemon.
func Shutdown() { Default.Shutdown() }

// AutoStart launches a managed daemon when the local API is not already up.
// Errors are ignored (caller may check Reachable() afterward).
func AutoStart() {
	if Reachable() {
		return
	}
	if err := Default.Start(); err != nil {
		return
	}
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if Reachable() {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Stop terminates the daemon started by this manager.
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.managed || m.cmd == nil || m.cmd.Process == nil {
		m.mu.Unlock()
		return fmt.Errorf("IPFS daemon is not running under DCS")
	}
	proc := m.cmd.Process
	m.mu.Unlock()

	if err := proc.Kill(); err != nil {
		return err
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !m.Managed() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for IPFS daemon to stop")
}

func (m *Manager) wait(cmd *exec.Cmd) {
	err := cmd.Wait()

	m.mu.Lock()
	m.cmd = nil
	m.managed = false
	onExit := m.onExit
	m.mu.Unlock()

	if onExit != nil {
		onExit(err)
	}
}

func resolveBinary() (string, error) {
	if p := strings.TrimSpace(os.Getenv("IPFS_BIN")); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("IPFS_BIN %q: %w", p, err)
		}
		return p, nil
	}
	bin, err := exec.LookPath("ipfs")
	if err == nil {
		return bin, nil
	}
	if runtime.GOOS == "windows" {
		bin, err = exec.LookPath("ipfs.exe")
	}
	if err != nil {
		return "", fmt.Errorf("ipfs not found on PATH (set IPFS_BIN to the full path to ipfs.exe): %w", err)
	}
	return bin, nil
}

func applyPlatformAttrs(cmd *exec.Cmd) {
	if runtime.GOOS != "windows" {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func discard(r io.Reader) {
	_, _ = io.Copy(io.Discard, r)
}
