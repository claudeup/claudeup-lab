package assets

import (
	"strings"
	"testing"
)

func TestDockerfileScriptsAreEmbedded(t *testing.T) {
	// Every COPY in the Dockerfile that references a .sh file must have
	// a corresponding embedded variable so buildFallback can write it
	// into the Docker build context.
	scripts := map[string][]byte{
		"init-claude-config.sh": InitClaudeConfig,
		"init-config-repo.sh":   InitConfigRepo,
		"init-claudeup.sh":      InitClaudeup,
		"init-firewall.sh":      InitFirewall,
	}

	dockerfile := string(Dockerfile)

	for name, content := range scripts {
		if !strings.Contains(dockerfile, name) {
			t.Errorf("Dockerfile does not reference %s", name)
		}
		if len(content) == 0 {
			t.Errorf("embedded script %s is empty", name)
		}
	}
}
