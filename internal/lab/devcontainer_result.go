package lab

import (
	"encoding/json"
	"fmt"
	"strings"
)

// devcontainerResult represents the JSON output from `devcontainer up`.
// The CLI emits this as the last line of stdout.
type devcontainerResult struct {
	Outcome     string `json:"outcome"`
	Message     string `json:"message,omitempty"`
	Description string `json:"description,omitempty"`
	ContainerID string `json:"containerId,omitempty"`
}

// parseDevcontainerResult extracts and parses the JSON result line from
// devcontainer CLI output. The result is always the last non-empty line
// of stdout, formatted as a JSON object with an "outcome" field.
func parseDevcontainerResult(output string) (*devcontainerResult, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("devcontainer produced no output")
	}

	// The JSON result is the last non-empty line
	lines := strings.Split(output, "\n")
	var jsonLine string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && strings.HasPrefix(line, "{") {
			jsonLine = line
			break
		}
	}

	if jsonLine == "" {
		return nil, fmt.Errorf("devcontainer produced no JSON result line")
	}

	var result devcontainerResult
	if err := json.Unmarshal([]byte(jsonLine), &result); err != nil {
		return nil, fmt.Errorf("parse devcontainer result: %w", err)
	}

	if result.Outcome == "error" {
		msg := result.Message
		if result.Description != "" {
			msg = fmt.Sprintf("%s: %s", msg, result.Description)
		}
		return &result, &PostCreateError{Detail: msg}
	}

	return &result, nil
}

// PostCreateError indicates the container started but postCreateCommand failed.
type PostCreateError struct {
	Detail string
}

func (e *PostCreateError) Error() string {
	return fmt.Sprintf("postCreateCommand failed: %s", e.Detail)
}
