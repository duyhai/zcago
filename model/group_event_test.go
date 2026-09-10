package model

import "testing"

func TestNewGroupEvent_JoinRequest(t *testing.T) {
	data := TGroupEventJoinRequest{GID: "g1", UIDs: []string{"u9"}, TotalPending: 1}

	ev := NewGroupEvent("me", "join_request", data)
	if ev.Type() != GroupEventTypeJoinRequest {
		t.Fatalf("Type = %q, want %q", ev.Type(), GroupEventTypeJoinRequest)
	}
	if ev.ThreadID() != "g1" || ev.Action() != "join_request" {
		t.Fatalf("ThreadID/Action = %q/%q", ev.ThreadID(), ev.Action())
	}
	if ev.IsSelf() {
		t.Fatal("IsSelf = true for a request from another account")
	}
	if !NewGroupEvent("u9", "join_request", data).IsSelf() {
		t.Fatal("IsSelf = false for a request that lists the account itself")
	}
}

func TestNewGroupEvent_DefaultCaseToleratesOtherPayloadTypes(t *testing.T) {
	// An action the model does not know lands in the default branch; the
	// payload there must not be assumed to be TGroupEventBase.
	ev := NewGroupEvent("me", "not_a_known_action", TGroupEventJoinRequest{GID: "g1"})
	if ev.Type() != GroupEventTypeUnknown || ev.ThreadID() != "g1" || ev.IsSelf() {
		t.Fatalf("unexpected event: type=%q thread=%q self=%v", ev.Type(), ev.ThreadID(), ev.IsSelf())
	}
}

func TestNewGroupEvent_BaseSelfDetection(t *testing.T) {
	bySource := NewGroupEvent("me", "update", TGroupEventBase{GID: "g1", SourceID: "me"})
	if !bySource.IsSelf() {
		t.Fatal("IsSelf = false when the account is the source")
	}
	byMember := NewGroupEvent("me", "leave", TGroupEventBase{
		GID: "g1", SourceID: "admin", UpdateMembers: []GroupEventUpdateMember{{ID: "me"}},
	})
	if !byMember.IsSelf() {
		t.Fatal("IsSelf = false when the account is an updated member")
	}
}
