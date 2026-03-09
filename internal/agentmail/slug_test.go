package agentmail

import "testing"

func TestProjectSlugFromPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/Users/jemanuel/projects/ntm", "users-jemanuel-projects-ntm"},
		{"/home/user/code/my-project", "home-user-code-my-project"},
		{"/var/www/html/site_v1", "var-www-html-site_v1"},
		{"/tmp/test project", "tmp-test-project"},
		{"/path/to/UPPERCASE", "path-to-uppercase"},
		{"/path/to/mixed-CASE_project", "path-to-mixed-case_project"},
		{"/root", "root"},
		{".", "root"},
		{"/", "root"},
		{"", ""},
		{"/path/with/!@#$%^&*()", "path-with"},
		{"/path/to/valid-123_ok", "path-to-valid-123_ok"},
		{"relative/path", "relative-path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ProjectSlugFromPath(tt.input)
			if got != tt.expected {
				t.Errorf("ProjectSlugFromPath(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}
