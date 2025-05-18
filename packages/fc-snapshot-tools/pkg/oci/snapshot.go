// Package oci provides functionality for pushing snapshots to OCI registries
package oci

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

const (
	// FirecrackerArtifactType is the artifact type for Firecracker snapshots
	FirecrackerArtifactType = "application/vnd.firecracker.layer.v1"
)

// PushSnapshot pushes a snapshot to an OCI registry using the ORAS tool
func PushSnapshot(ctx context.Context, ociRef, memFile, vmstateFile, configFile string) error {
	// Check if ORAS is installed
	if _, err := exec.LookPath("oras"); err != nil {
		return fmt.Errorf("oras command not found, please install it: %w", err)
	}

	// Check if files exist
	for _, file := range []string{memFile, vmstateFile, configFile} {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("file not found: %s: %w", file, err)
		}
	}

	// Run ORAS command to push the snapshot
	cmd := exec.CommandContext(
		ctx,
		"oras",
		"push",
		ociRef,
		"--artifact-type", FirecrackerArtifactType,
		memFile, vmstateFile, configFile,
	)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push snapshot: %s: %w", output, err)
	}

	return nil
}

// PullSnapshot pulls a snapshot from an OCI registry using the ORAS tool
func PullSnapshot(ctx context.Context, ociRef, outDir string) error {
	// Check if ORAS is installed
	if _, err := exec.LookPath("oras"); err != nil {
		return fmt.Errorf("oras command not found, please install it: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run ORAS command to pull the snapshot
	cmd := exec.CommandContext(
		ctx,
		"oras",
		"pull",
		ociRef,
		"--output", outDir,
	)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull snapshot: %s: %w", output, err)
	}

	return nil
}
