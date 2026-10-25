package main

import "testing"

func TestParseRunArgsAllowSecretWrite(t *testing.T) {
	opts := parseRunArgs([]string{
		"--allow-secret-write=entry",
		"--allow-secret-write", "./lib/auth.ca",
		"main.ca",
	})
	if len(opts.allowSecretWrite) != 2 {
		t.Fatalf("expected 2 allowed modules, got %v", opts.allowSecretWrite)
	}
	if opts.allowSecretWrite[0] != "entry" || opts.allowSecretWrite[1] != "./lib/auth.ca" {
		t.Errorf("unexpected allowlist: %v", opts.allowSecretWrite)
	}
	if opts.file != "main.ca" {
		t.Errorf("expected file main.ca, got %q", opts.file)
	}
}
