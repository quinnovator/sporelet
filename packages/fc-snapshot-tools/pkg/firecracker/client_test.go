package firecracker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Test the overall StartVM -> WaitForVSockHandshake -> CreateSnapshot workflow
func TestClientWorkflow(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	handshake := func(ctx context.Context) error {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/ready", nil)
		resp, err := srv.Client().Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}
	c, err := NewClient("fc", "jailer", "testvm", "", WithHTTPClient(srv.Client()), WithBaseURL(srv.URL), WithStartFunc(func(context.Context) error { return nil }), WithHandshakeFunc(handshake))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	cfg := VMConfig{
		KernelImagePath:   "kernel",
		RootDrive:         Drive{PathOnHost: "rootfs", IsRootDevice: true},
		KernelArgs:        "console=ttyS0",
		MemSizeMB:         64,
		VCPUCount:         1,
		NetworkInterfaces: []NetworkInterface{{HostDevName: "tap0", MacAddress: "AA"}},
	}

	if err := c.StartVM(context.Background(), cfg); err != nil {
		t.Fatalf("StartVM: %v", err)
	}

	if err := c.WaitForVSockHandshake(context.Background()); err != nil {
		t.Fatalf("Handshake: %v", err)
	}

	tmp := t.TempDir()
	snap := SnapshotConfig{
		MemFilePath:     filepath.Join(tmp, "mem"),
		VMStateFilePath: filepath.Join(tmp, "vm"),
		ConfigFilePath:  filepath.Join(tmp, "cfg"),
	}
	if err := c.CreateSnapshot(context.Background(), snap); err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}

	expected := []string{"/boot-source", "/drives/rootfs", "/machine-config", "/network-interfaces/eth0", "/actions", "/snapshot/create"}
	for _, e := range expected {
		found := false
		for _, p := range paths {
			if p == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected request to %s", e)
		}
	}
}

// Test restoring a snapshot triggers the correct API calls
func TestRestoreSnapshot(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	handshake := func(ctx context.Context) error {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/ready", nil)
		resp, err := srv.Client().Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}

	c, err := NewClient("fc", "jailer", "vm", "", WithHTTPClient(srv.Client()), WithBaseURL(srv.URL), WithStartFunc(func(context.Context) error { return nil }), WithHandshakeFunc(handshake))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tmp := t.TempDir()
	mem := filepath.Join(tmp, "mem")
	vm := filepath.Join(tmp, "vm")
	cfg := filepath.Join(tmp, "cfg")
	for _, f := range []string{mem, vm, cfg} {
		if err := os.WriteFile(f, []byte("dummy"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := c.RestoreSnapshot(context.Background(), RestoreConfig{MemFilePath: mem, VMStateFilePath: vm, ConfigFilePath: cfg}); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}

	if err := c.WaitForVSockHandshake(context.Background()); err != nil {
		t.Fatalf("Handshake: %v", err)
	}

	expected := []string{"/snapshot/load"}
	for _, e := range expected {
		found := false
		for _, p := range paths {
			if p == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected request to %s", e)
		}
	}
}
