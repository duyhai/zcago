package events

import (
	"encoding/json"
	"testing"

	"github.com/amrakk/zcago/model"
)

func TestActionData_UnmarshalJSON(t *testing.T) {
	t.Run("empty input does not panic", func(t *testing.T) {
		var d actionData
		if err := d.UnmarshalJSON(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("string-wrapped typing", func(t *testing.T) {
		var d actionData
		if err := json.Unmarshal([]byte(`"{\"uid\":\"u1\",\"ts\":\"1\",\"isPC\":0}"`), &d); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Typing.UID != "u1" || d.GroupTyping != (model.TGroupTyping{}) {
			t.Fatalf("decoded %+v", d)
		}
	})

	t.Run("group typing object", func(t *testing.T) {
		var d actionData
		if err := json.Unmarshal([]byte(`{"uid":"u1","gid":"g1"}`), &d); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.GroupTyping.GID != "g1" || d.GroupTyping.UID != "u1" {
			t.Fatalf("decoded %+v", d)
		}
	})
}
