package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func defaultBaseDir() string {
	return filepath.Join(os.Getenv("HOME"), ".claudeup-lab")
}

// baseDirFn returns the base directory for lab data. It defaults to
// defaultBaseDir and can be replaced in tests to use a temp directory.
var baseDirFn = defaultBaseDir

func resolveLab(resolver *lab.Resolver, name string) (*lab.Metadata, error) {
	if name != "" {
		return resolver.Resolve(name)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	return resolver.ResolveByCWD(cwd)
}
