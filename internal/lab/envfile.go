package lab

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseEnvFile reads a Docker-style env file and returns key-value pairs.
// Lines starting with # are comments. Blank lines are skipped.
// Each non-comment line must be KEY=VALUE format.
func ParseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("env file not found: %s", path)
		}
		return nil, fmt.Errorf("open env file %s: %w", path, err)
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip blank lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		idx := strings.IndexByte(trimmed, '=')
		if idx < 0 {
			return nil, fmt.Errorf("invalid line in env file %s (line %d): expected KEY=VALUE", path, lineNum)
		}

		key := trimmed[:idx]
		if key == "" {
			return nil, fmt.Errorf("empty key in env file %s (line %d): %q", path, lineNum, trimmed)
		}

		value := trimmed[idx+1:]
		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}

	return env, nil
}
