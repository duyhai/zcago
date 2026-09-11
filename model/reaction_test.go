package model

import (
	"encoding/json"
	"testing"
)

// decodeRefs unmarshals a ReactionContent and returns it, so the tests below
// exercise ReactionMessageRef.UnmarshalJSON through the same path the
// listener uses (content -> ReactionContent -> []ReactionMessageRef).
func decodeContent(t *testing.T, s string) (ReactionContent, error) {
	t.Helper()
	var c ReactionContent
	err := json.Unmarshal([]byte(s), &c)
	return c, err
}

func TestReactionRefNumericIDs(t *testing.T) {
	c, err := decodeContent(t, `{"rMsg":[{"gMsgID":12345,"cMsgID":999,"msgType":1}],"rIcon":"/-heart"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.RMsg) != 1 {
		t.Fatalf("rMsg len = %d, want 1", len(c.RMsg))
	}
	if c.RMsg[0].GMsgID != 12345 || c.RMsg[0].CMsgID != 999 || c.RMsg[0].MsgType != 1 {
		t.Fatalf("ref = %+v, want {12345 999 1}", c.RMsg[0])
	}
	if c.RIcon != ReactionHeart {
		t.Fatalf("icon = %q, want %q", c.RIcon, ReactionHeart)
	}
}

// The regression this patch exists for: zca-js types these ids as strings.
// A quoted id used to fail `int` unmarshalling and kill the whole frame.
func TestReactionRefStringIDs(t *testing.T) {
	c, err := decodeContent(t, `{"rMsg":[{"gMsgID":"12345","cMsgID":"999","msgType":"1"}],"rIcon":"/-heart"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.RMsg) != 1 {
		t.Fatalf("rMsg len = %d, want 1", len(c.RMsg))
	}
	if c.RMsg[0].GMsgID != 12345 || c.RMsg[0].CMsgID != 999 || c.RMsg[0].MsgType != 1 {
		t.Fatalf("ref = %+v, want {12345 999 1}", c.RMsg[0])
	}
}

func TestReactionRefHugeID(t *testing.T) {
	const huge = 9068274523746316453
	c, err := decodeContent(t, `{"rMsg":[{"gMsgID":9068274523746316453,"cMsgID":"9068274523746316453"}],"rIcon":"/-heart"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.RMsg[0].GMsgID != huge {
		t.Fatalf("gMsgID = %d, want %d", c.RMsg[0].GMsgID, huge)
	}
	if c.RMsg[0].CMsgID != huge {
		t.Fatalf("cMsgID = %d, want %d", c.RMsg[0].CMsgID, huge)
	}
}

// A garbage id must degrade to 0 (= "no target", which callers handle) and
// never take the frame down with it.
func TestReactionRefGarbageStringIsZeroNotError(t *testing.T) {
	for _, raw := range []string{
		`{"rMsg":[{"gMsgID":"not-a-number","cMsgID":""}],"rIcon":"/-heart"}`,
		`{"rMsg":[{"gMsgID":null,"cMsgID":true}],"rIcon":"/-heart"}`,
		`{"rMsg":[{"gMsgID":"99999999999999999999999"}],"rIcon":"/-heart"}`,
	} {
		c, err := decodeContent(t, raw)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", raw, err)
		}
		if len(c.RMsg) != 1 {
			t.Fatalf("%s: rMsg len = %d, want 1", raw, len(c.RMsg))
		}
		if c.RMsg[0].GMsgID != 0 || c.RMsg[0].CMsgID != 0 {
			t.Fatalf("%s: ref = %+v, want zero ids", raw, c.RMsg[0])
		}
		if c.RIcon != ReactionHeart {
			t.Fatalf("%s: icon = %q, want the icon to still decode", raw, c.RIcon)
		}
	}
}

func TestReactionRefAbsentRMsg(t *testing.T) {
	c, err := decodeContent(t, `{"rIcon":"/-heart"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.RMsg) != 0 {
		t.Fatalf("rMsg len = %d, want 0", len(c.RMsg))
	}
	if c.RIcon != ReactionHeart {
		t.Fatalf("icon = %q, want %q", c.RIcon, ReactionHeart)
	}
}

// Structural surprises still error: an object where an array belongs means
// the payload is not the shape we think it is.
func TestReactionRefObjectInsteadOfArrayStillErrors(t *testing.T) {
	if _, err := decodeContent(t, `{"rMsg":{"gMsgID":12345},"rIcon":"/-heart"}`); err == nil {
		t.Fatal("want an error for rMsg as an object, got nil")
	}
	if _, err := decodeContent(t, `{"rMsg":[12345],"rIcon":"/-heart"}`); err == nil {
		t.Fatal("want an error for a non-object rMsg entry, got nil")
	}
}
