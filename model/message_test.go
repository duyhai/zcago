package model

import (
	"encoding/json"
	"testing"
)

func decodeQuote(t *testing.T, s string) TQuote {
	t.Helper()
	var q TQuote
	if err := json.Unmarshal([]byte(s), &q); err != nil {
		t.Fatalf("unmarshal %s: %v", s, err)
	}
	return q
}

func TestQuoteStringAndNumberIDsDecodeIdentically(t *testing.T) {
	str := decodeQuote(t, `{"ownerId":"1234567890123456789","cliMsgId":"1726500000123","globalMsgId":"7400000000000000001","cliMsgType":1,"ts":"1726500000456","msg":"hi","attach":"","fromD":"A","ttl":0}`)
	num := decodeQuote(t, `{"ownerId":1234567890123456789,"cliMsgId":1726500000123,"globalMsgId":7400000000000000001,"cliMsgType":1,"ts":1726500000456,"msg":"hi","attach":"","fromD":"A","ttl":0}`)

	want := TQuote{
		OwnerID:     "1234567890123456789",
		CliMsgID:    1726500000123,
		GlobalMsgID: 7400000000000000001,
		CliMsgType:  1,
		Timestamp:   1726500000456,
		Msg:         "hi",
		FromD:       "A",
	}
	if str != want {
		t.Fatalf("string-shaped quote = %+v, want %+v", str, want)
	}
	if num != want {
		t.Fatalf("number-shaped quote = %+v, want %+v", num, want)
	}
}

func TestQuoteUnusableIDsBecomeZeroWithoutError(t *testing.T) {
	for _, v := range []string{`"abc"`, `""`, `null`, `true`, `1.5`, `"1e3"`, `{}`, `[]`, `"99999999999999999999"`} {
		q := decodeQuote(t, `{"ownerId":`+v+`,"cliMsgId":`+v+`,"globalMsgId":`+v+`,"ts":`+v+`,"msg":"kept"}`)
		if q.CliMsgID != 0 || q.GlobalMsgID != 0 || q.Timestamp != 0 {
			t.Errorf("%s: ids = %d/%d/%d, want 0", v, q.CliMsgID, q.GlobalMsgID, q.Timestamp)
		}
		if v != `"abc"` && v != `"1e3"` && v != `"99999999999999999999"` && q.OwnerID != "" {
			t.Errorf("%s: ownerId = %q, want empty", v, q.OwnerID)
		}
		if q.Msg != "kept" {
			t.Errorf("%s: msg = %q, other fields must still decode", v, q.Msg)
		}
	}
	// Absent fields are simply zero.
	if q := decodeQuote(t, `{}`); q != (TQuote{}) {
		t.Fatalf("empty quote = %+v, want zero", q)
	}
}

func TestQuoteStructuralMismatchStillErrors(t *testing.T) {
	for _, s := range []string{`[]`, `"x"`, `{"cliMsgType":"1"}`} {
		var q TQuote
		if err := json.Unmarshal([]byte(s), &q); err == nil {
			t.Errorf("%s: want error, got %+v", s, q)
		}
	}
}

func TestGroupMessageWithStringQuoteIDDecodes(t *testing.T) {
	var m TGroupMessage
	err := json.Unmarshal([]byte(`{"msgId":"m1","cliMsgId":"c1","uidFrom":"u1","idTo":"g1","ts":"1","content":"reply","quote":{"ownerId":"u2","cliMsgId":"1726500000123","globalMsgId":7400000000000000001,"cliMsgType":1,"ts":"1726500000456","msg":"orig"},"mentions":[]}`), &m)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Quote == nil || m.Quote.CliMsgID != 1726500000123 || m.Quote.GlobalMsgID != 7400000000000000001 || m.Quote.Timestamp != 1726500000456 {
		t.Fatalf("quote = %+v", m.Quote)
	}
}

func TestDeletedContentStringIDs(t *testing.T) {
	var c Content
	if err := json.Unmarshal([]byte(`[{"type":1,"actionType":2,"uidFrom":"11","uidTo":22,"clientDelMsgId":"33","globalDelMsgId":"x","destId":null}]`), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := TDeletedContent{Type: 1, ActionType: 2, UIDFrom: 11, UIDTo: 22, ClientDelMsgId: 33}
	if len(c.DeletedContent) != 1 || c.DeletedContent[0] != want {
		t.Fatalf("deleted content = %+v, want [%+v]", c.DeletedContent, want)
	}
}
