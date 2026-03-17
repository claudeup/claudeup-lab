package assets

import (
	"regexp"
	"testing"
)

func TestDockerfileScriptsAreEmbedded(t *testing.T) {
	// Every COPY in the Dockerfile that references a .sh file must have
	// a corresponding embedded variable so buildFallback can write it
	// into the Docker build context.
	embedded := map[string][]byte{
		"init-claude-config.sh": InitClaudeConfig,
		"init-config-repo.sh":   InitConfigRepo,
		"init-claudeup.sh":      InitClaudeup,
		"init-firewall.sh":      InitFirewall,
	}

	// Parse Dockerfile for all COPY directives referencing .sh files.
	// This catches regressions where a new script is added to the
	// Dockerfile but not to the embedded assets or this map.
	re := regexp.MustCompile(`(?m)^COPY\s+\S*\s+([\w.-]+\.sh)\s`)
	matches := re.FindAllStringSubmatch(string(Dockerfile), -1)
	if len(matches) == 0 {
		t.Fatal("no COPY *.sh directives found in Dockerfile")
	}

	for _, m := range matches {
		name := m[1]
		content, ok := embedded[name]
		if !ok {
			t.Errorf("Dockerfile COPYs %s but it is not in the embedded map (add //go:embed and update buildFallback)", name)
			continue
		}
		if len(content) == 0 {
			t.Errorf("embedded script %s is empty", name)
		}
	}
}
