package homedir

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func patchEnv(key, value string) func() {
	bck, exists := os.LookupEnv(key)
	deferFunc := func() {
		if exists {
			os.Setenv(key, bck)
		} else {
			os.Unsetenv(key)
		}
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
	defer patchEnv("USERPROFILE", "/custom/path/")()
	expected := filepath.Join("/", "custom", "path", "foo/bar")
	actual, err := Expand("~/foo/bar")

	if err != nil {
		t.Errorf("No error is expected, got: %v", err)
	} else if actual != expected {
		t.Errorf("Expected: %v; actual: %v", expected, actual)
	}
}

func TestDirWindows(t *testing.T) {
	t.Run("USERPROFILE prioritized over HOME and HOMEDRIVE/HOMEPATH", func(t *testing.T) {
		defer patchEnv("USERPROFILE", `C:\Users\profile`)()
		defer patchEnv("HOME", `C:\Users\home`)()
		defer patchEnv("HOMEDRIVE", `D:`)()
		defer patchEnv("HOMEPATH", `\Users\other`)()

		dir, err := dirWindows()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if expected := `C:\Users\profile`; dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	})

	t.Run("HOMEDRIVE and HOMEPATH prioritized over HOME", func(t *testing.T) {
		defer patchEnv("USERPROFILE", "")()
		defer patchEnv("HOMEDRIVE", `C:`)()
		defer patchEnv("HOMEPATH", `\Users\drivepath`)()
		defer patchEnv("HOME", `C:\Users\home`)()

		dir, err := dirWindows()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if expected := `C:\Users\drivepath`; dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	})

	t.Run("HOME fallback when system locations blank", func(t *testing.T) {
		defer patchEnv("USERPROFILE", "")()
		defer patchEnv("HOMEDRIVE", "")()
		defer patchEnv("HOMEPATH", "")()
		defer patchEnv("HOME", `C:\Users\home`)()

		dir, err := dirWindows()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if expected := `C:\Users\home`; dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	})

	t.Run("incomplete HOMEDRIVE falls back to HOME", func(t *testing.T) {
		defer patchEnv("USERPROFILE", "")()
		defer patchEnv("HOMEDRIVE", `C:`)()
		defer patchEnv("HOMEPATH", "")()
		defer patchEnv("HOME", `C:\Users\home`)()

		dir, err := dirWindows()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if expected := `C:\Users\home`; dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	})

	t.Run("incomplete HOMEPATH falls back to HOME", func(t *testing.T) {
		defer patchEnv("USERPROFILE", "")()
		defer patchEnv("HOMEDRIVE", "")()
		defer patchEnv("HOMEPATH", `\Users\pathonly`)()
		defer patchEnv("HOME", `C:\Users\home`)()

		dir, err := dirWindows()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if expected := `C:\Users\home`; dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	})

	t.Run("error when all blank", func(t *testing.T) {
		defer patchEnv("USERPROFILE", "")()
		defer patchEnv("HOMEDRIVE", "")()
		defer patchEnv("HOMEPATH", "")()
		defer patchEnv("HOME", "")()

		dir, err := dirWindows()
		if err == nil {
			t.Fatalf("expected error, got dir: %q", dir)
		}
	})
}

