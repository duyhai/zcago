package model

import (
	"encoding/json"

	"github.com/amrakk/zcago/config"
	"github.com/amrakk/zcago/errs"
)

type Urgency int

const (
	UrgDefault Urgency = iota
	UrgImportant
	UrgUrgent
)

type Message interface {
	Type() ThreadType
	ThreadID() string
	IsSelf() bool
}

type UserMessage struct {
	typ      ThreadType
	Data     TMessage
	threadID string
	isSelf   bool
}

func NewUserMessage(uid string, data TMessage) UserMessage {
	msg := UserMessage{
		typ:      ThreadTypeUser,
		Data:     data,
		threadID: data.UIDFrom,
		isSelf:   data.UIDFrom == config.DefaultUIDSelf,
	}

	if data.UIDFrom == config.DefaultUIDSelf {
		msg.threadID = data.IDTo
	}
	if data.IDTo == config.DefaultUIDSelf {
		msg.Data.IDTo = uid
	}
	if data.UIDFrom == config.DefaultUIDSelf {
		msg.Data.UIDFrom = uid
	}

	return msg
}

func (m UserMessage) Type() ThreadType { return m.typ }
func (m UserMessage) ThreadID() string { return m.threadID }
func (m UserMessage) IsSelf() bool     { return m.isSelf }

type GroupMessage struct {
	typ      ThreadType
	Data     TGroupMessage
	threadID string
	isSelf   bool
}

func NewGroupMessage(uid string, data TGroupMessage) GroupMessage {
	g := GroupMessage{
		typ:      ThreadTypeGroup,
		Data:     data,
		threadID: data.IDTo,
		isSelf:   data.UIDFrom == config.DefaultUIDSelf,
	}

	if data.UIDFrom == config.DefaultUIDSelf {
		g.Data.UIDFrom = uid
	}

	return g
}

func (m GroupMessage) Type() ThreadType { return m.typ }
func (m GroupMessage) ThreadID() string { return m.threadID }
func (m GroupMessage) IsSelf() bool     { return m.isSelf }

type OldMessages struct {
	Messages   []Message
	ThreadType ThreadType
}

func NewOldMessage(messages []Message, threadType ThreadType) OldMessages {
	return OldMessages{
		Messages:   messages,
		ThreadType: threadType,
	}
}

type TMessage struct {
	ActionID          string       `json:"actionId"`
	MsgID             string       `json:"msgId"`
	CliMsgID          string       `json:"cliMsgId"`
	MsgType           string       `json:"msgType"`
	UIDFrom           string       `json:"uidFrom"`
	IDTo              string       `json:"idTo"`
	DName             string       `json:"dName"`
	TS                string       `json:"ts"`
	Status            int          `json:"status"`
	Content           Content      `json:"content"`
	Notify            string       `json:"notify"`
	TTL               int          `json:"ttl"`
	UserID            string       `json:"userId"`
	UIN               string       `json:"uin"`
	TopOut            string       `json:"topOut"`
	TopOutTimeOut     string       `json:"topOutTimeOut"`
	TopOutImprTimeOut string       `json:"topOutImprTimeOut"`
	PropertyExt       *PropertyExt `json:"propertyExt,omitempty"`
	ParamsExt         ParamsExt    `json:"paramsExt"`
	CMD               int          `json:"cmd"`
	ST                int          `json:"st"`
	AT                int          `json:"at"`
	RealMsgID         string       `json:"realMsgId"`
	Quote             *TQuote      `json:"quote,omitempty"`
}

type TGroupMessage struct {
	TMessage
	Mentions []*TMention `json:"mentions,omitempty"`
}

type PropertyExt struct {
	Color   int    `json:"color"`
	Size    int    `json:"size"`
	Type    int    `json:"type"`
	SubType int    `json:"subType"`
	Ext     string `json:"ext"`
}

type ParamsExt struct {
	CountUnread  int `json:"countUnread"`
	ContainType  int `json:"containType"`
	PlatformType int `json:"platformType"`
}

type TQuote struct {
	OwnerID     string `json:"ownerId"`
	CliMsgID    int64  `json:"cliMsgId"`
	GlobalMsgID int64  `json:"globalMsgId"`
	CliMsgType  int    `json:"cliMsgType"`
	Timestamp   int64  `json:"ts"`
	Msg         string `json:"msg"`
	Attach      string `json:"attach"`
	FromD       string `json:"fromD"`
	TTL         uint   `json:"ttl"`
}

// UnmarshalJSON accepts the quote's id-like and timestamp-like fields --
// ownerId, cliMsgId, globalMsgId and ts -- as EITHER a JSON number or a
// quoted numeric string. zca-js types the numeric ones as numbers, but live
// group catch-up pages carry quotes whose cliMsgId is a string; decoding that
// into int64 failed the ENTIRE old-messages page (every message in it lost)
// and, on the live path, silently dropped the quoting message.
//
// An absent, null, empty or unparseable value leaves the field at its zero
// value and is NOT an error. cliMsgType and ttl stay strict: they are not
// ids, and there is no evidence of them arriving as strings. A quote that is
// not a JSON object at all still errors.
func (tq *TQuote) UnmarshalJSON(data []byte) error {
	type alias TQuote
	type quoteWire struct {
		OwnerID     json.RawMessage `json:"ownerId"`
		CliMsgID    json.RawMessage `json:"cliMsgId"`
		GlobalMsgID json.RawMessage `json:"globalMsgId"`
		Timestamp   json.RawMessage `json:"ts"`
		*alias
	}
	aux := quoteWire{alias: (*alias)(tq)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	tq.OwnerID = flexIDString(aux.OwnerID)
	tq.CliMsgID = flexInt64(aux.CliMsgID)
	tq.GlobalMsgID = flexInt64(aux.GlobalMsgID)
	tq.Timestamp = flexInt64(aux.Timestamp)
	return nil
}

type TAttachmentContent struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Href        string `json:"href"`
	Thumb       string `json:"thumb"`
	ChildNumber int    `json:"childnumber"`
	Action      string `json:"action"`
	Params      string `json:"params"`
	Type        string `json:"type"`
}

type TDeletedContent struct {
	Type           int `json:"type"`
	ActionType     int `json:"actionType"`
	UIDFrom        int `json:"uidFrom"`
	UIDTo          int `json:"uidTo"`
	ClientDelMsgId int `json:"clientDelMsgId"`
	GlobalDelMsgId int `json:"globalDelMsgId"`
	DestId         int `json:"destId"`
}

// UnmarshalJSON accepts the user/message id fields (uidFrom, uidTo,
// clientDelMsgId, globalDelMsgId, destId) as either a JSON number or a quoted
// numeric string; unusable values become 0 (see flexInt64). Before this a
// string id here failed DeletedContent, then TOtherContent (an array is not a
// map), so Content -- and with it the whole message page -- failed. type and
// actionType are enums and stay strict.
func (d *TDeletedContent) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type           int             `json:"type"`
		ActionType     int             `json:"actionType"`
		UIDFrom        json.RawMessage `json:"uidFrom"`
		UIDTo          json.RawMessage `json:"uidTo"`
		ClientDelMsgId json.RawMessage `json:"clientDelMsgId"`
		GlobalDelMsgId json.RawMessage `json:"globalDelMsgId"`
		DestId         json.RawMessage `json:"destId"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*d = TDeletedContent{
		Type:           raw.Type,
		ActionType:     raw.ActionType,
		UIDFrom:        flexInt(raw.UIDFrom),
		UIDTo:          flexInt(raw.UIDTo),
		ClientDelMsgId: flexInt(raw.ClientDelMsgId),
		GlobalDelMsgId: flexInt(raw.GlobalDelMsgId),
		DestId:         flexInt(raw.DestId),
	}
	return nil
}

type TOtherContent map[string]any

type MentionType int

const (
	MentionEach MentionType = iota
	MentionAll

	MentionAllUID = "-1"
)

type TMention struct {
	UID  string      `json:"uid"` // User ID being mentioned, or "-1" for mention all
	Pos  int         `json:"pos"` // Mention position
	Len  int         `json:"len"` // Mention length
	Type MentionType `json:"type"`
}

func (m *TMention) IsValid() bool {
	if m.Type == 1 && m.UID == MentionAllUID {
		return true
	}
	if m.Type == 0 && m.UID != "" && m.UID != MentionAllUID && m.Len > 0 {
		return true
	}
	return false
}

type Content struct {
	String         *string
	Attachment     *TAttachmentContent
	DeletedContent []TDeletedContent
	Other          TOtherContent
	//    []TOtherContent
}

func (c *Content) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.String = &s
		return nil
	}

	var attach TAttachmentContent
	if err := json.Unmarshal(data, &attach); err == nil && attach.Title != "" {
		c.Attachment = &attach
		return nil
	}

	var deletedContent []TDeletedContent
	if err := json.Unmarshal(data, &deletedContent); err == nil {
		c.DeletedContent = deletedContent
		return nil
	}

	var other TOtherContent
	if err := json.Unmarshal(data, &other); err == nil {
		c.Other = other
		return nil
	}

	return errs.NewZCA("Content: data did not match any known content type", "Content.UnmarshalJSON")
}

func (c Content) MarshalJSON() ([]byte, error) {
	if c.String != nil {
		return json.Marshal(c.String)
	}
	if c.Attachment != nil {
		return json.Marshal(c.Attachment)
	}
	if c.DeletedContent != nil {
		return json.Marshal(c.DeletedContent)
	}
	if c.Other != nil {
		return json.Marshal(c.Other)
	}
	return []byte("null"), nil
}

type OutboundMessage struct {
	MsgID    string
	CliMsgID string
	UIDFrom  string
	IDTo     string
	MsgType  string
	ST       int
	AT       int
	CMD      int
	TS       int64
}
