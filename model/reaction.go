package model

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/amrakk/zcago/config"
)

type ReactionIcon string

const (
	ReactionHeart       ReactionIcon = `/-heart`
	ReactionLike        ReactionIcon = `/-strong`
	ReactionHaha        ReactionIcon = `:>`
	ReactionWow         ReactionIcon = `:o`
	ReactionCry         ReactionIcon = `:-((`
	ReactionAngry       ReactionIcon = `:-h`
	ReactionKiss        ReactionIcon = `:-*`
	ReactionTearsOfJoy  ReactionIcon = `:\')`
	ReactionShit        ReactionIcon = `/-shit`
	ReactionRose        ReactionIcon = `/-rose`
	ReactionBrokenHeart ReactionIcon = `/-break`
	ReactionDislike     ReactionIcon = `/-weak`
	ReactionLove        ReactionIcon = `;xx`
	ReactionConfused    ReactionIcon = `;-/`
	ReactionWink        ReactionIcon = `;-)`
	ReactionFade        ReactionIcon = `/-fade`
	ReactionSun         ReactionIcon = `/-li`
	ReactionBirthday    ReactionIcon = `/-bd`
	ReactionBomb        ReactionIcon = `/-bome`
	ReactionOK          ReactionIcon = `/-ok`
	ReactionPeace       ReactionIcon = `/-v`
	ReactionThanks      ReactionIcon = `/-thanks`
	ReactionPunch       ReactionIcon = `/-punch`
	ReactionShare       ReactionIcon = `/-share`
	ReactionPray        ReactionIcon = `_()_`
	ReactionNo          ReactionIcon = `/-no`
	ReactionBad         ReactionIcon = `/-bad`
	ReactionLoveYou     ReactionIcon = `/-loveu`
	ReactionSad         ReactionIcon = `--b`
	ReactionVerySad     ReactionIcon = `:((`
	ReactionCool        ReactionIcon = `x-)`
	ReactionNerd        ReactionIcon = `8-)`
	ReactionBigSmile    ReactionIcon = `;-d`
	ReactionSunglasses  ReactionIcon = `b-)`
	ReactionNeutral     ReactionIcon = `:--|`
	ReactionSadFace     ReactionIcon = `p-(`
	ReactionBye         ReactionIcon = `:-bye`
	ReactionSleepy      ReactionIcon = `|-)`
	ReactionWipe        ReactionIcon = `:wipe`
	ReactionDig         ReactionIcon = `:-dig`
	ReactionAnguish     ReactionIcon = `&-(`
	ReactionHandclap    ReactionIcon = `:handclap`
	ReactionAngryFace   ReactionIcon = `>-|`
	ReactionFChair      ReactionIcon = `:-f`
	ReactionLChair      ReactionIcon = `:-l`
	ReactionRChair      ReactionIcon = `:-r`
	ReactionSilent      ReactionIcon = `;-x`
	ReactionSurprise    ReactionIcon = `:-o`
	ReactionEmbarrassed ReactionIcon = `;-s`
	ReactionAfraid      ReactionIcon = `;-a`
	ReactionSad2        ReactionIcon = `:-<`
	ReactionBigLaugh    ReactionIcon = `:))`
	ReactionRich        ReactionIcon = `$-)`
	ReactionBeer        ReactionIcon = `/-beer`
	ReactionNone        ReactionIcon = ``

	// more...

	DefaultReactionSource = 6
)

var reactionType = map[ReactionIcon]int{
	ReactionHaha:        0,
	ReactionLike:        3,
	ReactionHeart:       5,
	ReactionWow:         32,
	ReactionCry:         2,
	ReactionAngry:       20,
	ReactionKiss:        8,
	ReactionTearsOfJoy:  7,
	ReactionShit:        66,
	ReactionRose:        120,
	ReactionBrokenHeart: 65,
	ReactionDislike:     4,
	ReactionLove:        29,
	ReactionConfused:    51,
	ReactionWink:        45,
	ReactionFade:        121,
	ReactionSun:         67,
	ReactionBirthday:    126,
	ReactionBomb:        127,
	ReactionOK:          68,
	ReactionPeace:       69,
	ReactionThanks:      70,
	ReactionPunch:       71,
	ReactionShare:       72,
	ReactionPray:        73,
	ReactionNo:          131,
	ReactionBad:         132,
	ReactionLoveYou:     133,
	ReactionSad:         1,
	ReactionVerySad:     16,
	ReactionCool:        21,
	ReactionNerd:        22,
	ReactionBigSmile:    23,
	ReactionSunglasses:  26,
	ReactionNeutral:     30,
	ReactionSadFace:     35,
	ReactionBye:         36,
	ReactionSleepy:      38,
	ReactionWipe:        39,
	ReactionDig:         42,
	ReactionAnguish:     44,
	ReactionHandclap:    46,
	ReactionAngryFace:   47,
	ReactionFChair:      48,
	ReactionLChair:      49,
	ReactionRChair:      50,
	ReactionSilent:      52,
	ReactionSurprise:    53,
	ReactionEmbarrassed: 54,
	ReactionAfraid:      60,
	ReactionSad2:        61,
	ReactionBigLaugh:    62,
	ReactionRich:        63,
	ReactionBeer:        99,
}

func (icon ReactionIcon) TypeCode() int {
	if t, ok := reactionType[icon]; ok {
		return t
	}
	return -1
}

type Reaction struct {
	Type     ThreadType
	Data     TReaction
	ThreadID string
	IsSelf   bool
}

func NewReaction(uid string, data TReaction, threadType ThreadType) Reaction {
	if data.IDTo == config.DefaultUIDSelf {
		data.IDTo = uid
	}
	if data.UIDFrom == config.DefaultUIDSelf {
		data.UIDFrom = uid
	}

	isSelf := data.UIDFrom == config.DefaultUIDSelf

	threadID := data.UIDFrom
	if threadType == ThreadTypeGroup || isSelf {
		threadID = data.IDTo
	}

	return Reaction{
		Type:     threadType,
		Data:     data,
		ThreadID: threadID,
		IsSelf:   isSelf,
	}
}

type OldReactions struct {
	Reactions  []Reaction
	ThreadType ThreadType
}

func NewOldReactions(reactions []Reaction, threadType ThreadType) OldReactions {
	return OldReactions{
		Reactions:  reactions,
		ThreadType: threadType,
	}
}

type TReaction struct {
	ActionID string          `json:"actionId"`
	MsgID    string          `json:"msgId"`
	CliMsgID string          `json:"cliMsgId"`
	MsgType  string          `json:"msgType"`
	UIDFrom  string          `json:"uidFrom"`
	IDTo     string          `json:"idTo"`
	DName    *string         `json:"dName,omitempty"`
	Content  ReactionContent `json:"content"`
	TS       string          `json:"ts"`
	TTL      int             `json:"ttl"`

	// ContentRaw is the reaction's "content" field exactly as Zalo sent it,
	// retained BEFORE it is decoded into Content. Zalo has been observed to
	// vary the shape of content.rMsg (absent entirely, ids as numbers, ids
	// as quoted strings), and a decoded Content cannot tell those apart
	// after the fact -- a missing rMsg and an rMsg whose gMsgID is 0 both
	// come out as an unusable target. Callers that need to diagnose such an
	// event can read the wire shape here.
	//
	// It carries only ids, the icon code, rType and source -- never message
	// text -- and is set by the listener's reaction decoder; it is never
	// populated by a plain json.Unmarshal of TReaction (the `-` tag) and is
	// ignored when marshalling.
	ContentRaw string `json:"-"`
}

type ReactionData struct {
	RIcon  ReactionIcon `json:"rIcon"`
	RType  int          `json:"r.RType"`
	Source int          `json:"r.Source"`
}

func NewReactionData(icon ReactionIcon) ReactionData {
	return ReactionData{
		RIcon:  icon,
		Source: DefaultReactionSource,
		RType:  icon.TypeCode(),
	}
}

func (rid ReactionData) IsValid() bool {
	return len(rid.RIcon) != 0
}

type ReactionContent struct {
	RMsg []ReactionMessageRef `json:"rMsg"`
	ReactionData
}

type ReactionMessageRef struct {
	GMsgID  int `json:"gMsgID"`
	CMsgID  int `json:"cMsgID"`
	MsgType int `json:"msgType"`
}

// UnmarshalJSON accepts gMsgID/cMsgID/msgType as EITHER a JSON number or a
// quoted string. Zalo's own web client types these ids as strings
// (zca-js: `rMsg: { gMsgID: string; cMsgID: string; ... }[]`) while its
// addReaction sends them as numbers, so both shapes are on the wire. Before
// this, a quoted id failed to unmarshal into `int` and the error killed the
// WHOLE reaction frame, dropping every reaction in it rather than just the
// one field.
//
// A value that is neither a number nor a parseable integer string leaves the
// field at 0 and is NOT an error: an unusable id must degrade to "no target"
// (which callers already handle) instead of discarding the frame. Structural
// errors -- rMsg being an object instead of an array, a ref that is not a
// JSON object -- still error, because those mean the payload is not what we
// think it is at all.
func (r *ReactionMessageRef) UnmarshalJSON(data []byte) error {
	var raw struct {
		GMsgID  json.RawMessage `json:"gMsgID"`
		CMsgID  json.RawMessage `json:"cMsgID"`
		MsgType json.RawMessage `json:"msgType"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.GMsgID = flexInt(raw.GMsgID)
	r.CMsgID = flexInt(raw.CMsgID)
	r.MsgType = flexInt(raw.MsgType)
	return nil
}

// flexInt reads a JSON number or a quoted decimal string as an int,
// returning 0 for anything else (absent, null, empty, non-numeric, or out of
// range). It never reports an error -- see ReactionMessageRef.UnmarshalJSON.
func flexInt(b json.RawMessage) int {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return 0
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return 0
		}
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return 0
		}
		return int(n)
	}
	n, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return 0
	}
	return int(n)
}
