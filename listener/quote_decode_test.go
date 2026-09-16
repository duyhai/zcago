package listener

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/amrakk/zcago/listener/events"
	"github.com/amrakk/zcago/model"
)

// quotingGroupMsg is a group message quoting another one, with the quote's
// cliMsgId (and ts) as JSON strings -- the shape that failed live.
func quotingGroupMsg(msgID string) map[string]any {
	return map[string]any{
		"msgId": msgID, "cliMsgId": "c-" + msgID, "msgType": "webchat",
		"uidFrom": "u1", "idTo": "g1", "ts": "1726500000999", "content": "reply",
		"quote": map[string]any{
			"ownerId": "u2", "cliMsgId": "1726500000123", "globalMsgId": 7400000000000000001,
			"cliMsgType": 1, "ts": "1726500000456", "msg": "orig",
		},
	}
}

func plainGroupMsg(msgID string) map[string]any {
	return map[string]any{
		"msgId": msgID, "cliMsgId": "c-" + msgID, "msgType": "webchat",
		"uidFrom": "u3", "idTo": "g1", "ts": "1726500001000", "content": "plain",
	}
}

func TestOldMessagesGroupPageWithStringQuoteIDDecodes(t *testing.T) {
	raw, err := json.Marshal(map[string]any{"data": map[string]any{
		"groupMsgs": []any{plainGroupMsg("m1"), quotingGroupMsg("m2"), plainGroupMsg("m3")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := parseJSON[events.OldMessagesEventData](raw)
	if err != nil {
		t.Fatalf("decode old-messages page: %v", err)
	}
	if n := len(ev.Data.GroupMsgs); n != 3 {
		t.Fatalf("groupMsgs = %d, want 3", n)
	}
	q := ev.Data.GroupMsgs[1].Quote
	if q == nil || q.CliMsgID != 1726500000123 || q.Timestamp != 1726500000456 || q.OwnerID != "u2" {
		t.Fatalf("quote = %+v", q)
	}
}

func TestOldMessagesStructuralMismatchStillErrors(t *testing.T) {
	raw := []byte(`{"data":{"groupMsgs":{"msgId":"m1"}}}`)
	if _, err := parseJSON[events.OldMessagesEventData](raw); err == nil {
		t.Fatal("groupMsgs as an object must still fail to decode")
	}
}

// Through the real read loop: a 1_511_1 group catch-up frame containing a
// string-quoted message emits one OldMessages batch with every message.
func TestRun_OldGroupMessagesWithStringQuoteIDAreEmitted(t *testing.T) {
	cl := newFakeClient()
	ln, _, _ := startTestListener(t, cl, newTestSession("me"))

	cl.msgs <- eventFrame(t, 1, 511, 1, map[string]any{
		"groupMsgs": []any{quotingGroupMsg("m1"), plainGroupMsg("m2")},
	})

	select {
	case err := <-ln.Error():
		t.Fatalf("unexpected listener error: %v", err)
	case batch := <-ln.OldMessages():
		if batch.ThreadType != model.ThreadTypeGroup || len(batch.Messages) != 2 {
			t.Fatalf("batch = type %v, %d messages; want group, 2", batch.ThreadType, len(batch.Messages))
		}
		gm := batch.Messages[0].(model.GroupMessage)
		if gm.Data.Quote == nil || gm.Data.Quote.CliMsgID != 1726500000123 {
			t.Fatalf("quote = %+v", gm.Data.Quote)
		}
	case <-timeoutC():
		t.Fatal("timed out waiting for old messages")
	}
}

// Live path (1_521_0): groupMessageOrUndo swallows decode errors, so before
// the fix the quoting message was silently dropped. It must now be emitted.
func TestRun_LiveGroupMessageWithStringQuoteIDIsEmitted(t *testing.T) {
	cl := newFakeClient()
	ln, _, _ := startTestListener(t, cl, newTestSession("me"))

	cl.msgs <- eventFrame(t, 1, 521, 0, map[string]any{
		"groupMsgs": []any{quotingGroupMsg("m1")},
	})

	msg := recv(t, ln.Message(), "live group message")
	gm, ok := msg.(model.GroupMessage)
	if !ok || gm.Data.MsgID != "m1" || gm.Data.Quote == nil || gm.Data.Quote.CliMsgID != 1726500000123 {
		t.Fatalf("message = %+v", msg)
	}
}

func timeoutC() <-chan time.Time { return time.After(testTimeout) }
