// Package firecracker provides a client for interacting with the Firecracker API
package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

const afVsock = 40

type rawSockaddrVM struct {
	family    uint16
	reserved1 uint16
	port      uint32
	cid       uint32
	flags     uint8
	zero      [3]uint8
}

type sockaddrVM struct {
	cid   uint32
	port  uint32
	flags uint8
	raw   rawSockaddrVM
}

func (sa *sockaddrVM) ptr() unsafe.Pointer {
	sa.raw.family = afVsock
	sa.raw.cid = sa.cid
	sa.raw.port = sa.port
	sa.raw.flags = sa.flags
	return unsafe.Pointer(&sa.raw)
}

// Client represents a Firecracker API client
type Client struct {
	socketPath string
	fcBin      string
	jailerBin  string
	vmID       string
	cmd        *exec.Cmd
	httpClient *http.Client
	baseURL    string

	startFn     func(context.Context) error
	handshakeFn func(context.Context) error
}

// VMConfig represents the configuration for a Firecracker VM
type VMConfig struct {
	KernelImagePath   string
	RootDrive         Drive
	KernelArgs        string
	MemSizeMB         int
	VCPUCount         int
	NetworkInterfaces []NetworkInterface
}

// Drive represents a block device for a Firecracker VM
type Drive struct {
	PathOnHost   string
	IsReadOnly   bool
	IsRootDevice bool
}

// NetworkInterface represents a network interface for a Firecracker VM
type NetworkInterface struct {
	HostDevName string
	MacAddress  string
	IPAddress   string
	Netmask     string
	Gateway     string
}

// SnapshotConfig represents the configuration for creating a snapshot
type SnapshotConfig struct {
	MemFilePath     string
	VMStateFilePath string
	ConfigFilePath  string
}

// RestoreConfig represents the configuration for restoring from a snapshot
type RestoreConfig struct {
	MemFilePath     string
	VMStateFilePath string
	ConfigFilePath  string
}

// NewClient creates a new Firecracker client
// ClientOption configures optional settings for Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client for API communication.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = hc }
}

// WithBaseURL overrides the default API base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.baseURL = url }
}

// WithStartFunc overrides the function used to launch the Firecracker process.
func WithStartFunc(fn func(context.Context) error) ClientOption {
	return func(c *Client) { c.startFn = fn }
}

// WithHandshakeFunc overrides the vsock handshake check function.
func WithHandshakeFunc(fn func(context.Context) error) ClientOption {
	return func(c *Client) { c.handshakeFn = fn }
}

// NewClient creates a new Firecracker client
func NewClient(fcBin, jailerBin, vmID, socketPath string, opts ...ClientOption) (*Client, error) {
	if socketPath == "" {
		// Generate a unique socket path if not provided
		tmpDir, err := os.MkdirTemp("", "fc-socket-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory for socket: %w", err)
		}
		socketPath = filepath.Join(tmpDir, "firecracker.sock")
	}

	c := &Client{
		socketPath: socketPath,
		fcBin:      fcBin,
		jailerBin:  jailerBin,
		vmID:       vmID,
		baseURL:    "http://localhost",
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.handshakeFn == nil {
		c.handshakeFn = c.defaultHandshake
	}
	if c.startFn == nil {
		c.startFn = c.defaultStart
	}
	return c, nil
}

// StartVM starts a Firecracker VM with the given configuration
func (c *Client) StartVM(ctx context.Context, config VMConfig) error {
	// 1. Start Firecracker with jailer
	if err := c.startFirecracker(ctx); err != nil {
		return fmt.Errorf("failed to start Firecracker: %w", err)
	}

	// 2. Configure the VM
	if err := c.configureVM(ctx, config); err != nil {
		return fmt.Errorf("failed to configure VM: %w", err)
	}

	// 3. Start the VM
	if err := c.startInstance(ctx); err != nil {
		return fmt.Errorf("failed to start VM instance: %w", err)
	}

	return nil
}

// startFirecracker starts the Firecracker process with jailer
func (c *Client) startFirecracker(ctx context.Context) error {
	return c.startFn(ctx)
}

// configureVM configures the VM through the Firecracker API
func (c *Client) configureVM(ctx context.Context, config VMConfig) error {
	// Configure boot source
	bootSource := map[string]any{
		"kernel_image_path": config.KernelImagePath,
		"boot_args":         config.KernelArgs,
	}
	if err := c.apiPut(ctx, "/boot-source", bootSource); err != nil {
		return fmt.Errorf("failed to configure boot source: %w", err)
	}

	// Configure root drive
	drive := map[string]any{
		"drive_id":       "rootfs",
		"path_on_host":   config.RootDrive.PathOnHost,
		"is_root_device": config.RootDrive.IsRootDevice,
		"is_read_only":   config.RootDrive.IsReadOnly,
	}
	if err := c.apiPut(ctx, "/drives/rootfs", drive); err != nil {
		return fmt.Errorf("failed to configure root drive: %w", err)
	}

	// Configure machine
	machine := map[string]any{
		"vcpu_count":   config.VCPUCount,
		"mem_size_mib": config.MemSizeMB,
	}
	if err := c.apiPut(ctx, "/machine-config", machine); err != nil {
		return fmt.Errorf("failed to configure machine: %w", err)
	}

	// Configure network interfaces
	for i, netIf := range config.NetworkInterfaces {
		ifID := fmt.Sprintf("eth%d", i)
		netConfig := map[string]any{
			"iface_id":      ifID,
			"host_dev_name": netIf.HostDevName,
			"guest_mac":     netIf.MacAddress,
		}
		if err := c.apiPut(ctx, fmt.Sprintf("/network-interfaces/%s", ifID), netConfig); err != nil {
			return fmt.Errorf("failed to configure network interface %s: %w", ifID, err)
		}
	}

	return nil
}

// startInstance starts the VM instance
func (c *Client) startInstance(ctx context.Context) error {
	action := map[string]any{
		"action_type": "InstanceStart",
	}
	return c.apiPut(ctx, "/actions", action)
}

// WaitForVSockHandshake waits for the vsock handshake to complete
func (c *Client) WaitForVSockHandshake(ctx context.Context) error {
	return c.handshakeFn(ctx)
}

// defaultHandshake polls the guest agent ready endpoint until it reports ready.
func (c *Client) defaultHandshake(ctx context.Context) error {
	const (
		guestCID  = 3
		guestPort = 5005
	)
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		fd, err := syscall.Socket(afVsock, syscall.SOCK_STREAM, 0)
		if err != nil {
			return nil, err
		}
		sa := &sockaddrVM{cid: guestCID, port: guestPort}
		_, _, errno := syscall.RawSyscall(syscall.SYS_CONNECT, uintptr(fd), uintptr(sa.ptr()), unsafe.Sizeof(sa.raw))
		if errno != 0 {
			syscall.Close(fd)
			return nil, errno
		}
		f := os.NewFile(uintptr(fd), "vsock")
		conn, err := net.FileConn(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		return conn, nil
	}
	client := &http.Client{Transport: &http.Transport{DialContext: dial}}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://vsock/ready", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

// defaultStart launches Firecracker using the jailer binary and waits for the socket.
func (c *Client) defaultStart(ctx context.Context) error {
	c.cmd = exec.CommandContext(
		ctx,
		c.jailerBin,
		"--id", c.vmID,
		"--exec-file", c.fcBin,
		"--uid", "0",
		"--gid", "0",
		"--chroot-base-dir", "/tmp",
		"--",
		"--api-sock", c.socketPath,
	)

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Firecracker process: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()

	for i := 0; i < 100; i++ {
		if _, err := os.Stat(c.socketPath); err == nil {
			return nil
		}
		select {
		case err := <-done:
			return fmt.Errorf("firecracker exited prematurely: %w", err)
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("timed out waiting for Firecracker socket")
}

// CreateSnapshot creates a snapshot of the VM
func (c *Client) CreateSnapshot(ctx context.Context, config SnapshotConfig) error {
	snapshot := map[string]any{
		"mem_file_path": config.MemFilePath,
		"snapshot_type": "Full",
		"snapshot_path": config.VMStateFilePath,
		"version":       "1.0.0",
	}

	if err := c.apiPut(ctx, "/snapshot/create", snapshot); err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Save VM configuration to a file
	vmConfig, err := c.getVMConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VM config: %w", err)
	}

	if err := os.WriteFile(config.ConfigFilePath, vmConfig, 0644); err != nil {
		return fmt.Errorf("failed to write VM config: %w", err)
	}

	return nil
}

// RestoreSnapshot loads a snapshot and resumes the VM
func (c *Client) RestoreSnapshot(ctx context.Context, config RestoreConfig) error {
	if err := c.startFirecracker(ctx); err != nil {
		return fmt.Errorf("failed to start Firecracker: %w", err)
	}

	load := map[string]any{
		"snapshot_path": config.VMStateFilePath,
		"mem_file_path": config.MemFilePath,
		"resume_vm":     true,
	}

	if err := c.apiPut(ctx, "/snapshot/load", load); err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	return nil
}

// getVMConfig gets the VM configuration
func (c *Client) getVMConfig(ctx context.Context) ([]byte, error) {
	var machine map[string]any
	if err := c.apiGetJSON(ctx, "/machine-config", &machine); err != nil {
		return nil, fmt.Errorf("get machine-config: %w", err)
	}

	var boot map[string]any
	if err := c.apiGetJSON(ctx, "/boot-source", &boot); err != nil {
		return nil, fmt.Errorf("get boot-source: %w", err)
	}

	var rootfs map[string]any
	if err := c.apiGetJSON(ctx, "/drives/rootfs", &rootfs); err != nil {
		return nil, fmt.Errorf("get rootfs: %w", err)
	}

	cfg := map[string]any{
		"machine-config": machine,
		"boot-source":    boot,
		"rootfs":         rootfs,
	}

	return json.Marshal(cfg)
}

// apiPut sends a PUT request to the Firecracker API
func (c *Client) apiPut(ctx context.Context, path string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		fmt.Sprintf("%s%s", c.baseURL, path),
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, body)
	}

	return nil
}

// apiGet sends a GET request to the Firecracker API and returns the status code.
func (c *Client) apiGet(ctx context.Context, path string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s", c.baseURL, path), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// apiGetJSON sends a GET request and decodes the JSON response into v.
func (c *Client) apiGetJSON(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s", c.baseURL, path), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, body)
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

// Cleanup cleans up resources used by the client
func (c *Client) Cleanup() error {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}
