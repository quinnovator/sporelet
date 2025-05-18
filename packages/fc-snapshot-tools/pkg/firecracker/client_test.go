package firecracker

import (
	"context"
	"testing"
)

func TestNewClientGeneratesSocket(t *testing.T) {
	c, err := NewClient("firecracker", "jailer", "testvm", "")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if c.socketPath == "" {
		t.Fatal("expected socket path to be generated")
	}
}

func TestWaitForVSockHandshakeCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := &Client{}
	if err := c.WaitForVSockHandshake(ctx); err != context.Canceled {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}
