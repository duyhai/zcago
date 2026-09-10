package listener

import (
	"errors"
	"strings"
	"testing"

	"github.com/amrakk/zcago/internal/websocketx"
)

// panickingSession panics inside the handler that first asks for the
// account UID, standing in for any handler-level bug on a live frame.
func panickingSession(msg string) stubSession {
	return stubSession{uid: func() string { panic(msg) }}
}

// messageEvent is the 1_501_0 payload shape (events.MessageEventData).
func messageEvent(content string) map[string]any {
	return map[string]any{"msgs": []map[string]any{{
		"msgId":   "1",
		"msgType": "webchat",
		"uidFrom": "u1",
		"idTo":    "me",
		"content": content,
	}}}
}

func TestRun_HandlerPanicIsReportedAndLoopContinues(t *testing.T) {
	cl := newFakeClient()
	ln, _, _ := startTestListener(t, cl, panickingSession("synthetic handler panic"))

	const payload = "private-message-body"
	cl.msgs <- eventFrame(t, 1, 501, 0, messageEvent(payload))

	err := recv(t, ln.Error(), "panic error")

	var hp *HandlerPanicError
	if !errors.As(err, &hp) {
		t.Fatalf("Error() delivered %T (%v), want a HandlerPanicError cause", err, err)
	}
	if hp.Key != "1_501_0" {
		t.Fatalf("Key = %q, want %q", hp.Key, "1_501_0")
	}
	if hp.Value != "synthetic handler panic" {
		t.Fatalf("Value = %v, want the panic value", hp.Value)
	}
	if len(hp.Stack) == 0 || !strings.Contains(string(hp.Stack), "handleMessages") {
		t.Fatalf("Stack does not point at the panicking handler:\n%s", hp.Stack)
	}
	for _, s := range []string{err.Error(), string(hp.Stack)} {
		if strings.Contains(s, payload) {
			t.Fatalf("panic report leaks the frame payload: %s", s)
		}
	}
	if !strings.Contains(err.Error(), "1_501_0") || !strings.Contains(err.Error(), "synthetic handler panic") {
		t.Fatalf("Error() text lacks key or panic value: %s", err)
	}

	// The loop must still be alive: a frame that does not touch the
	// session is handled normally.
	cl.msgs <- eventFrame(t, 1, 502, 0, deliveredEvent(
		map[string]any{"msgId": "m2", "deliveredUids": []string{"u2"}},
	))
	batch := recv(t, ln.DeliveredMessages(), "delivered batch after panic")
	if len(batch) != 1 || batch[0].ThreadID() != "u2" {
		t.Fatalf("delivered batch after panic = %+v", batch)
	}
}

func TestRun_PanicDoesNotMaskClose(t *testing.T) {
	cl := newFakeClient()
	ln, _, done := startTestListener(t, cl, panickingSession("boom"))

	cl.msgs <- eventFrame(t, 1, 501, 0, messageEvent("x"))
	recv(t, ln.Error(), "panic error")

	cl.closed <- websocketx.CloseInfo{Code: ZaloManualClosure}

	ci := recv(t, ln.Closed(), "Closed() after a recovered panic")
	if ci.Code != ZaloManualClosure {
		t.Fatalf("Closed() code = %d, want %d", ci.Code, ZaloManualClosure)
	}
	waitClosed(t, done, "run to return after close")
}

func TestRun_ContextCancelStopsLoop(t *testing.T) {
	cl := newFakeClient()
	ln, cancel, done := startTestListener(t, cl, panickingSession("boom"))

	cl.msgs <- eventFrame(t, 1, 501, 0, messageEvent("x"))
	recv(t, ln.Error(), "panic error")

	cancel()
	waitClosed(t, done, "run to return after context cancel")
}
