package model

import "testing"

func TestNewUserDeliveredMessage_EmptyDeliveredUIDs(t *testing.T) {
	for name, uids := range map[string][]string{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			dm := NewUserDeliveredMessage(TDeliveredMessage{MsgID: "m1", DeliveredUIDs: uids})
			if got := dm.ThreadID(); got != "" {
				t.Fatalf("ThreadID = %q, want empty for a receipt with no recipients", got)
			}
			if dm.Type() != ThreadTypeUser || dm.IsSelf() {
				t.Fatalf("unexpected Type/IsSelf: %v/%v", dm.Type(), dm.IsSelf())
			}
			if dm.Data.MsgID != "m1" {
				t.Fatalf("Data not retained: %+v", dm.Data)
			}
		})
	}
}

func TestNewUserDeliveredMessage_FirstUIDIsThread(t *testing.T) {
	dm := NewUserDeliveredMessage(TDeliveredMessage{DeliveredUIDs: []string{"u1", "u2"}})
	if got := dm.ThreadID(); got != "u1" {
		t.Fatalf("ThreadID = %q, want %q", got, "u1")
	}
}
