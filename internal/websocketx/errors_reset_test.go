package websocketx

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
)

// A TCP reset on read reaches readLoop as net.OpError{Err: *os.SyscallError{Err: ECONNRESET}},
// wrapped by coder/websocket ("failed to get reader: failed to read frame header: ...").
// It must be fatal, or readLoop spins on the dead socket.
func TestIsConnClosedOrReset_WrappedSyscallError(t *testing.T) {
	for _, errno := range []syscall.Errno{syscall.ECONNRESET, syscall.EPIPE} {
		op := &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read", errno)}
		err := fmt.Errorf("failed to get reader: %w", fmt.Errorf("failed to read frame header: %w", op))
		if !isConnClosedOrReset(err) {
			t.Errorf("%v: want fatal, got non-fatal for %q", errno, err)
		}
	}
}
