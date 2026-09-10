package listener

import (
	"testing"

	"github.com/amrakk/zcago/model"
)

// deliveredEvent is the 1_502_0 payload shape (events.MessageStatusEventData).
func deliveredEvent(entries ...map[string]any) map[string]any {
	return map[string]any{"delivereds": entries}
}

func TestRun_EmptyDeliveredUIDsDoesNotPanic(t *testing.T) {
	cl := newFakeClient()
	ln, _, _ := startTestListener(t, cl, newTestSession("me"))

	// A receipt with no recipients, then a well-formed one. The loop is a
	// single goroutine and channels are FIFO, so if the first frame had
	// produced anything it would arrive before the second one's event.
	cl.msgs <- eventFrame(t, 1, 502, 0, deliveredEvent(
		map[string]any{"msgId": "m1", "deliveredUids": []string{}},
	))
	cl.msgs <- eventFrame(t, 1, 502, 0, deliveredEvent(
		map[string]any{"msgId": "m2", "deliveredUids": []string{"u2"}},
	))

	batch := recv(t, ln.DeliveredMessages(), "delivered batch")
	if len(batch) != 1 {
		t.Fatalf("got %d delivered messages, want 1", len(batch))
	}
	if got := batch[0].ThreadID(); got != "u2" {
		t.Fatalf("ThreadID = %q, want %q", got, "u2")
	}
	if got := batch[0].(model.UserDeliveredMessage).Data.MsgID; got != "m2" {
		t.Fatalf("MsgID = %q, want %q (the empty-uids receipt must be skipped)", got, "m2")
	}
}

func TestRun_AllEmptyDeliveredUIDsEmitsNothing(t *testing.T) {
	cl := newFakeClient()
	ln, _, _ := startTestListener(t, cl, newTestSession("me"))

	cl.msgs <- eventFrame(t, 1, 502, 0, deliveredEvent(
		map[string]any{"msgId": "m1", "deliveredUids": []string{}},
		map[string]any{"msgId": "m2"},
	))
	// A seen receipt behind it proves the frame above was fully processed.
	cl.msgs <- eventFrame(t, 1, 502, 0, map[string]any{
		"seens": []map[string]any{{"idTo": "u3", "msgId": "m3"}},
	})

	seen := recv(t, ln.SeenMessages(), "seen batch")
	if len(seen) != 1 || seen[0].ThreadID() != "u3" {
		t.Fatalf("seen batch = %+v, want one entry for u3", seen)
	}
	select {
	case batch := <-ln.DeliveredMessages():
		t.Fatalf("unexpected delivered batch %+v: receipts without recipients must not be emitted", batch)
	default:
	}
}
