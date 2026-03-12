package agentic

import (
	"fmt"
	"net"
	"testing"
)

func TestIsPortAvailable(t *testing.T) {
	// Test with a port that should be available (using 0 to get a random free port)
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to get a free port: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	freePort := addr.Port
	listener.Close()

	// Now test that the port is available
	available, err := isPortAvailable(fmt.Sprintf("%d", freePort))
	if err == nil && available {
		t.Logf("Port %d is available as expected", freePort)
	}

	// Test with a port that is in use
	listener2, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to get another free port: %v", err)
	}
	defer listener2.Close()

	addr2 := listener2.Addr().(*net.TCPAddr)
	usedPort := addr2.Port

	// This port should NOT be available
	available2, err2 := isPortAvailable(fmt.Sprintf("%d", usedPort))
	if available2 {
		t.Errorf("Port %d should not be available but was reported as available", usedPort)
	}
	if err2 == nil {
		t.Errorf("Expected an error when checking a port in use, but got nil")
	}
}
