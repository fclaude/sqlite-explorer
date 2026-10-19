package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestAppVersionMatchesReleaseMetadata(t *testing.T) {
	for _, tc := range []struct {
		path string
		keys []string
	}{
		{"wails.json", []string{"info", "productVersion"}},
		{"frontend/package.json", []string{"version"}},
		{"frontend/package-lock.json", []string{"version"}},
		{"frontend/package-lock.json", []string{"packages", "", "version"}},
	} {
		t.Run(tc.path+"/"+tc.keys[0], func(t *testing.T) {
			data, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			var value any
			if err := json.Unmarshal(data, &value); err != nil {
				t.Fatal(err)
			}
			for _, key := range tc.keys {
				object, ok := value.(map[string]any)
				if !ok {
					t.Fatalf("missing version metadata at %q", key)
				}
				value = object[key]
			}
			if value != version {
				t.Errorf("release version = %v, binary version = %s", value, version)
			}
		})
	}
}

func TestVersionOutputIdentifiesPackageWithoutStartingDesktop(t *testing.T) {
	got, handled := versionOutput([]string{"--version"}, "0.2.0", "abc123", "7")
	if !handled || got != "sqlite-explorer 0.2.0 (main build 7; commit abc123)" {
		t.Fatalf("version output = %q, handled = %v", got, handled)
	}
}

func TestVersionOutputPreservesNormalLaunchArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"database.sqlite"}, {"--version", "database.sqlite"}} {
		if output, handled := versionOutput(args, "0.2.0", "abc123", "7"); handled || output != "" {
			t.Fatalf("args %q unexpectedly handled: %q", args, output)
		}
	}
}
