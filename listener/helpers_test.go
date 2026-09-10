package listener

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/amrakk/zcago/internal/websocketx"
	"github.com/amrakk/zcago/session"
	"github.com/coder/websocket"
)

// fakeClient is a websocketx.Client whose inbound channels the test feeds
// directly, standing in for a live Zalo socket.
type fakeClient struct {
	msgs   chan websocketx.Message
	errs   chan error
	closed chan websocketx.CloseInfo
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		msgs:   make(chan websocketx.Message, 16),
		errs:   make(chan error, 4),
		closed: make(chan websocketx.CloseInfo, 1),
	}
}

func (f *fakeClient) Messages() <-chan websocketx.Message                        { return f.msgs }
func (f *fakeClient) Errors() <-chan error                                       { return f.errs }
func (f *fakeClient) Closed() <-chan websocketx.CloseInfo                        { return f.closed }
func (f *fakeClient) Write(context.Context, websocket.MessageType, []byte) error { return nil }
func (f *fakeClient) WriteText(context.Context, string) error                    { return nil }

func (f *fakeClient) Close(code int, reason string) {
	select {
	case f.closed <- websocketx.CloseInfo{Code: code, Reason: reason}:
	default:
	}
}

// stubSession satisfies session.MutableContext for the two methods the
// router reaches in these tests. Every other method dereferences the
// embedded nil interface and panics, which keeps the stub honest: a test
// that wanders into an unexpected call fails loudly.
type stubSession struct {
	session.MutableContext
	uid func() string
}

func (s stubSession) UID() string                   { return s.uid() }
func (s stubSession) WSPingInterval() time.Duration { return 0 }

func newTestSession(uid string) stubSession {
	return stubSession{uid: func() string { return uid }}
}

// startTestListener wires a listener around cl and runs its read loop
// exactly as Start does after a successful dial. The returned done channel
// closes when run returns.
func startTestListener(t *testing.T, cl *fakeClient, sc session.MutableContext) (*listener, context.CancelFunc, <-chan struct{}) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	ln := &listener{
		sc:         sc,
		ch:         initializeChannels(),
		client:     cl,
		selfListen: true,
		cancel:     cancel,
	}

	done := make(chan struct{})
	ln.wg.Add(1)
	go func() {
		defer close(done)
		ln.run(ctx, false)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	return ln, cancel, done
}

// eventFrame builds the binary frame the listener parses: a 4-byte header
// (version, little-endian cmd, sub) followed by the plaintext (encrypt=0)
// envelope whose "data" field carries {"data": event} as a JSON string.
func eventFrame(t *testing.T, version uint8, cmd uint16, sub uint8, event any) websocketx.Message {
	t.Helper()

	inner, err := json.Marshal(map[string]any{"data": event})
	if err != nil {
		t.Fatalf("marshal inner event: %v", err)
	}

	return envelopeFrame(t, version, cmd, sub, map[string]any{
		"encrypt": EncryptionTypeNone,
		"data":    string(inner),
	})
}

// cipherKeyFrame is the 1_1_1 frame the server sends right after connect.
func cipherKeyFrame(t *testing.T, key string) websocketx.Message {
	t.Helper()
	return envelopeFrame(t, 1, 1, 1, map[string]any{
		"key":     key,
		"encrypt": EncryptionTypeNone,
		"data":    "",
	})
}

func envelopeFrame(t *testing.T, version uint8, cmd uint16, sub uint8, envelope map[string]any) websocketx.Message {
	t.Helper()

	data, err := encodeFrame(WSPayload{Version: version, CMD: cmd, SubCMD: sub, Data: envelope})
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	return websocketx.Message{Type: websocketx.BinaryMessage, Data: data}
}

const testTimeout = 2 * time.Second

// recv waits for one value on ch or fails the test.
func recv[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

// waitClosed waits for done to close or fails the test.
func waitClosed(t *testing.T, done <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", what)
	}
}
