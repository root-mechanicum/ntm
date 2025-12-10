package status

import (
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ansi",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "color codes",
			input:    "\x1b[32mgreen\x1b[0m text",
			expected: "green text",
		},
		{
			name:     "multiple codes",
			input:    "\x1b[1m\x1b[34mbold blue\x1b[0m",
			expected: "bold blue",
		},
		{
			name:     "cursor movement",
			input:    "\x1b[2Jclear screen",
			expected: "clear screen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPromptLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		agentType string
		expected  bool
	}{
		// Claude prompts
		{name: "claude prompt lowercase", line: "claude>", agentType: "cc", expected: true},
		{name: "claude prompt with space", line: "claude> ", agentType: "cc", expected: true},
		{name: "Claude prompt uppercase", line: "Claude>", agentType: "cc", expected: true},

		// Codex prompts
		{name: "codex prompt", line: "codex>", agentType: "cod", expected: true},
		{name: "shell prompt for codex", line: "user@host:~$", agentType: "cod", expected: true},

		// Gemini prompts
		{name: "gemini prompt", line: "gemini>", agentType: "gmi", expected: true},
		{name: "Gemini prompt", line: "Gemini>", agentType: "gmi", expected: true},

		// User shell prompts
		{name: "dollar prompt", line: "user@host:~$ ", agentType: "user", expected: true},
		{name: "percent prompt", line: "user@host %", agentType: "user", expected: true},
		{name: "starship prompt", line: "~/project ❯", agentType: "user", expected: true},

		// Generic prompts
		{name: "generic > prompt", line: ">", agentType: "", expected: true},
		{name: "generic > prompt with space", line: "> ", agentType: "", expected: true},

		// Non-prompts
		{name: "regular text", line: "hello world", agentType: "cc", expected: false},
		{name: "empty string", line: "", agentType: "cc", expected: false},
		{name: "whitespace only", line: "   ", agentType: "cc", expected: false},

		// With ANSI codes
		{name: "prompt with ansi", line: "\x1b[32mclaude>\x1b[0m", agentType: "cc", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPromptLine(tt.line, tt.agentType)
			if result != tt.expected {
				t.Errorf("IsPromptLine(%q, %q) = %v, want %v", tt.line, tt.agentType, result, tt.expected)
			}
		})
	}
}

func TestDetectIdleFromOutput(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		agentType string
		expected  bool
	}{
		{
			name:      "claude idle at prompt",
			output:    "Some previous output\nMore text\nclaude>",
			agentType: "cc",
			expected:  true,
		},
		{
			name:      "claude working",
			output:    "Processing request...\nGenerating code...\n",
			agentType: "cc",
			expected:  false,
		},
		{
			name:      "claude prompt with trailing newlines",
			output:    "Output\nclaude>\n\n",
			agentType: "cc",
			expected:  true,
		},
		{
			name:      "codex at shell prompt",
			output:    "Command completed\nuser@host:~$",
			agentType: "cod",
			expected:  true,
		},
		{
			name:      "gemini idle",
			output:    "Response complete.\ngemini>",
			agentType: "gmi",
			expected:  true,
		},
		{
			name:      "empty output",
			output:    "",
			agentType: "cc",
			expected:  false,
		},
		{
			name:      "only whitespace",
			output:    "\n\n   \n",
			agentType: "cc",
			expected:  false,
		},
		{
			name:      "output with ansi codes",
			output:    "\x1b[32mSuccess!\x1b[0m\n\x1b[34mclaude>\x1b[0m",
			agentType: "cc",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectIdleFromOutput(tt.output, tt.agentType)
			if result != tt.expected {
				t.Errorf("DetectIdleFromOutput(%q, %q) = %v, want %v",
					tt.output, tt.agentType, result, tt.expected)
			}
		})
	}
}

func TestGetLastNonEmptyLine(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "simple output",
			output:   "line1\nline2\nline3",
			expected: "line3",
		},
		{
			name:     "trailing newlines",
			output:   "line1\nline2\n\n\n",
			expected: "line2",
		},
		{
			name:     "with ansi",
			output:   "\x1b[32mcolored\x1b[0m\n",
			expected: "colored",
		},
		{
			name:     "empty",
			output:   "",
			expected: "",
		},
		{
			name:     "only whitespace",
			output:   "   \n\t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLastNonEmptyLine(tt.output)
			if result != tt.expected {
				t.Errorf("GetLastNonEmptyLine(%q) = %q, want %q",
					tt.output, result, tt.expected)
			}
		})
	}
}
