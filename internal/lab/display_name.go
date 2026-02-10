package lab

import (
	"fmt"
	"regexp"
	"strings"
)

var validNameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func ComputeDisplayName(projectName, profile, userName string) string {
	if userName != "" {
		return userName
	}
	return projectName + "-" + profile
}

func ValidateDisplayName(name string) error {
	if name == "" {
		return fmt.Errorf("display name must not be empty")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("display name must not start with '.'")
	}
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("display name %q contains invalid characters (allowed: A-Z, a-z, 0-9, '.', '_', '-')", name)
	}
	return nil
}

func DisambiguateDisplayName(name, shortID string, existingNames map[string]bool) string {
	if !existingNames[name] {
		return name
	}
	return name + "-" + shortID
}
