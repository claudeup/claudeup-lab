package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client wraps Docker CLI commands for container and volume operations.
type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) IsRunning() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// FindContainer returns the container ID for a running devcontainer
// matching the given worktree path label, or empty string if none found.
func (c *Client) FindContainer(worktreePath string) (string, error) {
	cmd := exec.Command("docker", "ps", "-q",
		"--filter", fmt.Sprintf("label=devcontainer.local_folder=%s", worktreePath))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker ps: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// FindContainerIncludingStopped returns the container ID including stopped
// containers matching the given worktree path label.
func (c *Client) FindContainerIncludingStopped(worktreePath string) (string, error) {
	cmd := exec.Command("docker", "ps", "-aq",
		"--filter", fmt.Sprintf("label=devcontainer.local_folder=%s", worktreePath))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker ps: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) StopContainer(id string) error {
	cmd := exec.Command("docker", "stop", id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker stop %s: %w", id, err)
	}
	return nil
}

func (c *Client) RemoveContainer(id string) error {
	cmd := exec.Command("docker", "rm", "-f", id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker rm %s: %w", id, err)
	}
	return nil
}

// ListVolumes returns Docker volume names containing the given pattern.
func (c *Client) ListVolumes(pattern string) ([]string, error) {
	cmd := exec.Command("docker", "volume", "ls", "--format", "{{.Name}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker volume ls: %w", err)
	}

	var matches []string
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name != "" && strings.Contains(name, pattern) {
			matches = append(matches, name)
		}
	}
	return matches, nil
}

func (c *Client) RemoveVolumes(names []string) error {
	if len(names) == 0 {
		return nil
	}
	args := append([]string{"volume", "rm"}, names...)
	cmd := exec.Command("docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker volume rm: %w", err)
	}
	return nil
}

// ContainerHostname returns the hostname of a running devcontainer.
func (c *Client) ContainerHostname(worktreePath string) (string, error) {
	cmd := exec.Command("devcontainer", "exec",
		"--workspace-folder", worktreePath, "hostname")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get container hostname: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
