package homedir

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func patchEnv(key, value string) func() {
	bck := os.Getenv(key)
	deferFunc := func() {
		os.Setenv(key, bck)
	}

	if value != "" {
		os.Setenv(key, value)
	} else {
		os.Unsetenv(key)
	}

	return deferFunc
}

func BenchmarkDir(b *testing.B) {
	// We do this for any "warmups"
	for i := 0; i < 10; i++ {
		Dir()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Dir()
	}
}

func TestDir(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	dir, err := Dir()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if u.HomeDir != dir {
		t.Fatalf("%#v != %#v", u.HomeDir, dir)
	}

	DisableCache = true
	defer func() { DisableCache = false }()
	defer patchEnv("HOME", "")()
	dir, err = Dir()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if u.HomeDir != dir {
		t.Fatalf("%#v != %#v", u.HomeDir, dir)
	}
}

func TestExpand(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	cases := []struct {
		Input  string
		Output string
		Err    bool
	}{
		{
			"/foo",
			"/foo",
			false,
		},

		{
			"~/foo",
			filepath.Join(u.HomeDir, "foo"),
			false,
		},

		{
			"",
			"",
			false,
		},

		{
			"~",
			u.HomeDir,
			false,
		},

		{
			"~foo/foo",
			"",
			true,
		},
	}

	for _, tc := range cases {
		actual, err := Expand(tc.Input)
		if (err != nil) != tc.Err {
			t.Fatalf("Input: %#v\n\nErr: %s", tc.Input, err)
		}

		if actual != tc.Output {
			t.Fatalf("Input: %#v\n\nOutput: %#v", tc.Input, actual)
		}
	}

	DisableCache = true
	defer func() { DisableCache = false }()
	defer patchEnv("HOME", "/custom/path/")()
	expected := filepath.Join("/", "custom", "path", "foo/bar")
	actual, err := Expand("~/foo/bar")

	if err != nil {
		t.Errorf("No error is expected, got: %v", err)
	} else if actual != expected {
		t.Errorf("Expected: %v; actual: %v", expected, actual)
	}
}

func withEnv(vars map[string]string, fn func()) {
	saved := make(map[string]string)
	existed := make(map[string]bool)

	keys := []string{"HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "HOMESHARE", "SYSTEMDRIVE"}
	for _, k := range keys {
		val, ok := os.LookupEnv(k)
		existed[k] = ok
		if ok {
			saved[k] = val
		}
		os.Unsetenv(k)
	}

	for k, v := range vars {
		if v != "" {
			os.Setenv(k, v)
		} else {
			os.Unsetenv(k)
		}
	}

	defer func() {
		for _, k := range keys {
			if existed[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	fn()
}

func TestDirWindows(t *testing.T) {
	cases := []struct {
		name        string
		env         map[string]string
		expected    string
		expectError bool
	}{
		{
			name: "HOME takes precedence",
			env: map[string]string{
				"HOME":        `C:\Users\from_home`,
				"USERPROFILE": `C:\Users\from_userprofile`,
				"HOMEDRIVE":   `C:`,
				"HOMEPATH":    `\Users\from_homepath`,
			},
			expected: `C:\Users\from_home`,
		},
		{
			name: "USERPROFILE takes precedence over HOMEDRIVE and HOMEPATH",
			env: map[string]string{
				"USERPROFILE": `C:\Users\alice`,
				"HOMEDRIVE":   `C:`,
				"HOMEPATH":    `\Users\bob`,
				"HOMESHARE":   `\\server\share\users\carol`,
			},
			expected: `C:\Users\alice`,
		},
		{
			name: "HOMEDRIVE and HOMEPATH combined",
			env: map[string]string{
				"HOMEDRIVE": `C:`,
				"HOMEPATH":  `\Users\dave`,
			},
			expected: `C:\Users\dave`,
		},
		{
			name: "HOMEDRIVE without trailing slash and HOMEPATH without leading slash",
			env: map[string]string{
				"HOMEDRIVE": `D:`,
				"HOMEPATH":  `Users\eve`,
			},
			expected: `D:\Users\eve`,
		},
		{
			name: "HOMEDRIVE with trailing slash and HOMEPATH with leading slash",
			env: map[string]string{
				"HOMEDRIVE": `D:\`,
				"HOMEPATH":  `\Users\frank`,
			},
			expected: `D:\Users\frank`,
		},
		{
			name: "HOMESHARE and HOMEPATH fallback",
			env: map[string]string{
				"HOMESHARE": `\\server\share`,
				"HOMEPATH":  `\users\grace`,
			},
			expected: `\\server\share\users\grace`,
		},
		{
			name: "HOMESHARE alone fallback",
			env: map[string]string{
				"HOMESHARE": `\\server\share\users\heidi`,
			},
			expected: `\\server\share\users\heidi`,
		},
		{
			name: "HOMESHARE with root HOMEPATH",
			env: map[string]string{
				"HOMESHARE": `\\server\share\users\ivan`,
				"HOMEPATH":  `\`,
			},
			expected: `\\server\share\users\ivan`,
		},
		{
			name: "SYSTEMDRIVE and HOMEPATH fallback",
			env: map[string]string{
				"SYSTEMDRIVE": `C:`,
				"HOMEPATH":    `\Users\judy`,
			},
			expected: `C:\Users\judy`,
		},
		{
			name: "Absolute HOMEPATH with drive letter",
			env: map[string]string{
				"HOMEPATH": `C:\Users\mallory`,
			},
			expected: `C:\Users\mallory`,
		},
		{
			name: "Absolute HOMEPATH with UNC path",
			env: map[string]string{
				"HOMEPATH": `\\server\share\users\oscar`,
			},
			expected: `\\server\share\users\oscar`,
		},
		{
			name: "All variables blank returns error",
			env: map[string]string{
				"HOME":        "",
				"USERPROFILE": "",
				"HOMEDRIVE":   "",
				"HOMEPATH":    "",
				"HOMESHARE":   "",
				"SYSTEMDRIVE": "",
			},
			expectError: true,
		},
		{
			name: "HOMEDRIVE set without HOMEPATH returns error",
			env: map[string]string{
				"HOMEDRIVE": `C:`,
			},
			expectError: true,
		},
		{
			name: "Relative HOMEPATH alone returns error",
			env: map[string]string{
				"HOMEPATH": `\Users\peggy`,
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			withEnv(tc.env, func() {
				actual, err := dirWindows()
				if tc.expectError {
					if err == nil {
						t.Fatalf("expected error, got nil with result %q", actual)
					}
					if actual != "" {
						t.Fatalf("expected empty result on error, got %q", actual)
					}
					return
				}

				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if actual != tc.expected {
					t.Fatalf("expected %q, got %q", tc.expected, actual)
				}
			})
		})
	}
}

func TestJoinWindowsPath(t *testing.T) {
	cases := []struct {
		base     string
		sub      string
		expected string
	}{
		{"C:", `\Users\test`, `C:\Users\test`},
		{"C:", "Users/test", `C:\Users\test`},
		{`C:\`, `\Users\test`, `C:\Users\test`},
		{"C:", "", "C:"},
		{"C:", `\`, `C:\`},
		{`\\server\share`, `\users\test`, `\\server\share\users\test`},
		{`\\server\share`, "", `\\server\share`},
		{`\\server\share`, `\`, `\\server\share`},
	}

	for _, tc := range cases {
		actual := joinWindowsPath(tc.base, tc.sub)
		if actual != tc.expected {
			t.Errorf("joinWindowsPath(%q, %q) = %q, want %q", tc.base, tc.sub, actual, tc.expected)
		}
	}
}

func TestIsWindowsAbs(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{`C:\Users`, true},
		{`D:/Users`, true},
		{`\\server\share`, true},
		{`//server/share`, true},
		{`\Users`, false},
		{`/Users`, false},
		{`C:Users`, false},
		{"relative", false},
		{"", false},
	}

	for _, tc := range cases {
		actual := isWindowsAbs(tc.input)
		if actual != tc.expected {
			t.Errorf("isWindowsAbs(%q) = %v, want %v", tc.input, actual, tc.expected)
		}
	}
}
