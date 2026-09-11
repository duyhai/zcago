package events

import (
	"encoding/json"
	"testing"

	"github.com/amrakk/zcago/model"
)

// frame wraps a reaction `content` string into the reacts frame Zalo sends.
func frame(content string) string {
	b, _ := json.Marshal(content)
	return `{"reacts":[{"actionId":"1","msgId":"777","cliMsgId":"888","msgType":"webchat",` +
		`"uidFrom":"0","idTo":"123","ts":"1757563615000","ttl":0,"content":` + string(b) + `}]}`
}

func decodeFrame(t *testing.T, content string) (model.TReaction, error) {
	t.Helper()
	var d ReactionEventData
	if err := json.Unmarshal([]byte(frame(content)), &d); err != nil {
		return model.TReaction{}, err
	}
	if len(d.Reactions) != 1 {
		t.Fatalf("decoded %d reactions, want 1", len(d.Reactions))
	}
	return d.Reactions[0], nil
}

func TestReactionNumericTargetDecodes(t *testing.T) {
	const content = `{"rMsg":[{"gMsgID":12345,"cMsgID":999,"msgType":1}],"rIcon":"/-heart","rType":5,"source":6}`
	r, err := decodeFrame(t, content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Content.RMsg) != 1 || r.Content.RMsg[0].GMsgID != 12345 {
		t.Fatalf("content = %+v, want target 12345", r.Content)
	}
	if r.ContentRaw != content {
		t.Fatalf("ContentRaw = %q, want the raw wire string", r.ContentRaw)
	}
}

// The live bug: a string-shaped gMsgID used to make the WHOLE frame error,
// so every reaction in it was lost. It must now decode to a real target.
func TestReactionStringTargetNoLongerKillsTheFrame(t *testing.T) {
	r, err := decodeFrame(t, `{"rMsg":[{"gMsgID":"9068274523746316453","cMsgID":"123","msgType":1}],"rIcon":"/-heart"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Content.RMsg) != 1 || r.Content.RMsg[0].GMsgID != 9068274523746316453 {
		t.Fatalf("content = %+v, want the 19-digit target", r.Content)
	}
}

// The only shape that reproduces the live symptom (icon decodes, target
// empty, no error). ContentRaw is what makes it distinguishable downstream.
func TestReactionAbsentRMsgKeepsRawContent(t *testing.T) {
	const content = `{"rIcon":"/-heart","rType":5,"source":6}`
	r, err := decodeFrame(t, content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Content.RMsg) != 0 {
		t.Fatalf("rMsg len = %d, want 0", len(r.Content.RMsg))
	}
	if r.Content.RIcon != model.ReactionHeart {
		t.Fatalf("icon = %q, want the icon to still decode", r.Content.RIcon)
	}
	if r.ContentRaw != content {
		t.Fatalf("ContentRaw = %q, want %q", r.ContentRaw, content)
	}
}

func TestReactionObjectRMsgStillErrors(t *testing.T) {
	if _, err := decodeFrame(t, `{"rMsg":{"gMsgID":12345},"rIcon":"/-heart"}`); err == nil {
		t.Fatal("want an error for rMsg as an object, got nil")
	}
}

func TestReactionEmptyContentHasEmptyRaw(t *testing.T) {
	var d ReactionEventData
	raw := `{"reacts":[{"actionId":"1","msgId":"777","cliMsgId":"888","uidFrom":"0","idTo":"123","ts":"1","ttl":0,"content":""}]}`
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.Reactions) != 1 {
		t.Fatalf("decoded %d reactions, want 1", len(d.Reactions))
	}
	if d.Reactions[0].ContentRaw != "" {
		t.Fatalf("ContentRaw = %q, want empty", d.Reactions[0].ContentRaw)
	}
}
