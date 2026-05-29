package ipfsdaemon

import "testing"

func TestManager_notManagedInitially(t *testing.T) {
	m := New()
	if m.Managed() {
		t.Fatal("expected not managed")
	}
}

func TestManager_stopWhenNotManaged(t *testing.T) {
	m := New()
	if err := m.Stop(); err == nil {
		t.Fatal("expected error stopping idle daemon")
	}
}

func TestDefault_singleton(t *testing.T) {
	if Default == nil {
		t.Fatal("expected Default manager")
	}
}

func TestShutdown_idleNoOp(t *testing.T) {
	m := New()
	m.Shutdown()
	if m.Managed() {
		t.Fatal("expected not managed after idle shutdown")
	}
}

func TestResolveBinary_orSkip(t *testing.T) {
	_, err := resolveBinary()
	if err != nil {
		t.Skip("ipfs not on PATH in test environment")
	}
}
