package main

import "testing"

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
