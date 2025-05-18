package fc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPushSnapshot_MissingFile(t *testing.T) {
	dir := t.TempDir()
	mem := filepath.Join(dir, "snapshot.mem")
	os.WriteFile(mem, []byte("dummy"), 0644)

	vm := filepath.Join(dir, "snapshot.vmstate")
	cfg := filepath.Join(dir, "snapshot.config")

	if err := PushSnapshot(context.Background(), "ref", mem, vm, cfg); err == nil {
		t.Fatal("expected error when files are missing")
	}
}
