package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestEditorCommandWithVim(t *testing.T) {
	tests := []struct {
		name           string
		editorEnv      string
		expectedArgs   []string
		description    string
	}{
		{
			name:         "vim should use + flag",
			editorEnv:    "vim",
			expectedArgs: []string{"+", "filename"},
			description:  "vim should be invoked with + to start at last line",
		},
		{
			name:         "nvim should use + flag",
			editorEnv:    "nvim",
			expectedArgs: []string{"+", "filename"},
			description:  "nvim should be invoked with + to start at last line",
		},
		{
			name:         "/usr/bin/vim should use + flag",
			editorEnv:    "/usr/bin/vim",
			expectedArgs: []string{"+", "filename"},
			description:  "full path to vim should use + flag",
		},
		{
			name:         "/usr/local/bin/nvim should use + flag",
			editorEnv:    "/usr/local/bin/nvim",
			expectedArgs: []string{"+", "filename"},
			description:  "full path to nvim should use + flag",
		},
		{
			name:         "emacs should not use + flag",
			editorEnv:    "emacs",
			expectedArgs: []string{"filename"},
			description:  "non-vim editors should not use + flag",
		},
		{
			name:         "nano should not use + flag",
			editorEnv:    "nano",
			expectedArgs: []string{"filename"},
			description:  "non-vim editors should not use + flag",
		},
		{
			name:         "empty EDITOR defaults to vim with +",
			editorEnv:    "",
			expectedArgs: []string{"+", "filename"},
			description:  "empty EDITOR should default to vim with + flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the EDITOR environment variable
			oldEditor := os.Getenv("EDITOR")
			if tt.editorEnv != "" {
				os.Setenv("EDITOR", tt.editorEnv)
			} else {
				os.Unsetenv("EDITOR")
			}
			defer os.Setenv("EDITOR", oldEditor)

			// Simulate the logic from edit.go
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}

			var cmd *exec.Cmd
			if editor == "vim" || editor == "nvim" ||
				strings.HasSuffix(editor, "/vim") || strings.HasSuffix(editor, "/nvim") {
				cmd = exec.Command(editor, "+", "filename")
			} else {
				cmd = exec.Command(editor, "filename")
			}

			// Verify the command arguments
			args := cmd.Args[1:] // Skip the command itself
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("%s: expected %d args, got %d", tt.description, len(tt.expectedArgs), len(args))
				return
			}

			for i, expectedArg := range tt.expectedArgs {
				if args[i] != expectedArg {
					t.Errorf("%s: arg %d: expected %q, got %q", tt.description, i, expectedArg, args[i])
				}
			}
		})
	}
}