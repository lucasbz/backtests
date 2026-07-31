package main

import (
	"net"
	"testing"
)

func TestRun_NoCommand(t *testing.T) {
	if err := run([]string{}); err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	if err := run([]string{"nope"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRun_Serve_InvalidFlag(t *testing.T) {
	err := run([]string{"serve", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// TestRun_Serve_AddrInUse exercises the success path of serve()'s flag
// parsing and its call into http.ListenAndServe, without leaving a server
// running forever: it binds -addr to a port that's already taken, so
// ListenAndServe fails immediately with "address already in use" instead of
// blocking.
func TestRun_Serve_AddrInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer ln.Close()

	err = run([]string{"serve", "-addr", ln.Addr().String()})
	if err == nil {
		t.Fatal("expected error because the address is already in use")
	}
}
