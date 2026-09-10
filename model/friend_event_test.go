package model

import (
	"encoding/json"
	"testing"
)

func TestTFriendEventPinCreateTopicParams_UnmarshalJSON(t *testing.T) {
	t.Run("empty input does not panic", func(t *testing.T) {
		var p TFriendEventPinCreateTopicParams
		if err := p.UnmarshalJSON(nil); err == nil {
			t.Fatal("expected a decode error for empty input")
		}
	})

	t.Run("string-wrapped object", func(t *testing.T) {
		var p TFriendEventPinCreateTopicParams
		if err := json.Unmarshal([]byte(`"{\"senderUid\":\"u1\",\"title\":\"hi\"}"`), &p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.SenderUID != "u1" || p.Title != "hi" {
			t.Fatalf("decoded %+v", p)
		}
	})
}
