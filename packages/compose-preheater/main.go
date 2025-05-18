// Command compose-preheater launches Docker Compose services and waits until ready.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// containerState represents the state section from docker inspect
type containerState struct {
	Status string `json:"Status"`
	Health *struct {
		Status string `json:"Status"`
	} `json:"Health,omitempty"`
}

func run(ctx context.Context, composeFile string, timeout time.Duration) error {
	// 1. docker compose up -d
	upCmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "up", "-d")
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr
	if err := upCmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}

	// 2. Get container IDs
	psCmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "ps", "-q")
	out, err := psCmd.Output()
	if err != nil {
		return fmt.Errorf("docker compose ps: %w", err)
	}
	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		return fmt.Errorf("no containers found")
	}

	deadline := time.Now().Add(timeout)
	for {
		allReady := true
		for _, id := range ids {
			inspectCmd := exec.CommandContext(ctx, "docker", "inspect", id)
			iout, err := inspectCmd.Output()
			if err != nil {
				return fmt.Errorf("docker inspect %s: %w", id, err)
			}
			var arr []struct {
				State containerState `json:"State"`
			}
			if err := json.Unmarshal(iout, &arr); err != nil {
				return fmt.Errorf("decode inspect output: %w", err)
			}
			if len(arr) == 0 {
				return fmt.Errorf("docker inspect returned empty result")
			}
			st := arr[0].State
			if st.Status != "running" {
				allReady = false
				break
			}
			if st.Health != nil && st.Health.Status != "healthy" {
				allReady = false
				break
			}
		}
		if allReady {
			fmt.Println("services are ready")
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for services to be ready")
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	composeFile := flag.String("f", "docker-compose.yml", "compose file")
	timeout := flag.Duration("timeout", 60*time.Second, "wait timeout")
	flag.Parse()

	ctx := context.Background()
	if err := run(ctx, *composeFile, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
