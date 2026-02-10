package lab_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestComputeDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		profile  string
		userName string
		want     string
	}{
		{"default", "myapp", "base", "", "myapp-base"},
		{"custom name", "myapp", "base", "custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lab.ComputeDisplayName(tt.project, tt.profile, tt.userName)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"simple", "myapp-base", true},
		{"with dots", "my.app", true},
		{"with underscore", "my_app", true},
		{"starts with dot", ".hidden", false},
		{"double dot", "..", false},
		{"spaces", "my app", false},
		{"slashes", "my/app", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lab.ValidateDisplayName(tt.input)
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid, got nil error")
			}
		})
	}
}
