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

// controlEvent is the 1_601_0 payload shape (events.ControlEventData).
func controlEvent(actType, act string, data any) map[string]any {
	return map[string]any{"controls": []map[string]any{{
		"content": map[string]any{"act_type": actType, "act": act, "data": data},
	}}}
}

func TestRun_JoinRequestGroupEventIsEmitted(t *testing.T) {
	cl := newFakeClient()
	ln, _, _ := startTestListener(t, cl, newTestSession("me"))

	cl.msgs <- eventFrame(t, 1, 601, 0, controlEvent("group", "join_request", map[string]any{
		"groupId": "g1", "uids": []string{"u9"}, "totalPending": 1, "time": "1",
	}))

	ev := recv(t, ln.Group(), "group event")
	if ev.Type() != model.GroupEventTypeJoinRequest || ev.ThreadID() != "g1" {
		t.Fatalf("group event = type %q thread %q", ev.Type(), ev.ThreadID())
	}
	if _, ok := ev.Data().(model.TGroupEventJoinRequest); !ok {
		t.Fatalf("Data() is %T, want TGroupEventJoinRequest", ev.Data())
	}
	select {
	case err := <-ln.Error():
		t.Fatalf("unexpected error: %v", err)
	default:
	}
}
