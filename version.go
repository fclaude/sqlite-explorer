package main

import "fmt"

// Release builders may set these with -ldflags. A source build never claims a
// workstation package revision or source identity it was not given.
var version = "0.2.0"
var sourceCommit = "development"
var packageRevision = "development"

func versionOutput(args []string, version, commit, revision string) (string, bool) {
	if len(args) != 1 || args[0] != "--version" {
		return "", false
	}
	return fmt.Sprintf("sqlite-explorer %s (main build %s; commit %s)", version, revision, commit), true
}
