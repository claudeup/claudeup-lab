package lab

import (
	"errors"
	"strings"
	"testing"
)

func TestParseDevcontainerResult(t *testing.T) {
	tests := []struct {
		name              string
		output            string
		wantOutcome       string
		wantErr           bool
		wantPostCreateErr bool
		errContains       string
	}{
		{
			name:        "success outcome",
			output:      `{"outcome":"success","containerId":"abc123","remoteUser":"node"}`,
			wantOutcome: "success",
		},
		{
			name:        "success with preceding log lines",
			output:      "some log line\nanother log\n" + `{"outcome":"success","containerId":"abc123"}`,
			wantOutcome: "success",
		},
		{
			name:              "error outcome",
			output:            `{"outcome":"error","message":"postCreateCommand failed","description":"exit code 1"}`,
			wantOutcome:       "error",
			wantErr:           true,
			wantPostCreateErr: true,
			errContains:       "postCreateCommand failed",
		},
		{
			name:              "error outcome with preceding log lines",
			output:            "building image...\ninstalling deps...\n" + `{"outcome":"error","message":"script failed"}`,
			wantOutcome:       "error",
			wantErr:           true,
			wantPostCreateErr: true,
			errContains:       "script failed",
		},
		{
			name:        "empty output",
			output:      "",
			wantErr:     true,
			errContains: "no output",
		},
		{
			name:        "no JSON line",
			output:      "just some log output\nnothing json here",
			wantErr:     true,
			errContains: "no JSON result",
		},
		{
			name:        "trailing newline after JSON",
			output:      `{"outcome":"success","containerId":"abc123"}` + "\n",
			wantOutcome: "success",
		},
		{
			name:        "trailing whitespace after JSON",
			output:      `{"outcome":"success","containerId":"abc123"}` + "  \n",
			wantOutcome: "success",
		},
		{
			name:              "error with description",
			output:            `{"outcome":"error","message":"An error occurred setting up the container","description":"postCreateCommand failed with exit code 127"}`,
			wantOutcome:       "error",
			wantErr:           true,
			wantPostCreateErr: true,
			errContains:       "exit code 127",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDevcontainerResult(tt.output)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				if tt.wantPostCreateErr {
					var pce *PostCreateError
					if !errors.As(err, &pce) {
						t.Errorf("expected PostCreateError, got %T", err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Outcome != tt.wantOutcome {
				t.Errorf("outcome = %q, want %q", result.Outcome, tt.wantOutcome)
			}
		})
	}
}
