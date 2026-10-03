package model

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
)

// flexInt64 reads a JSON number or a quoted decimal string as an int64,
// returning 0 for anything else (absent, null, empty, non-numeric, fractional
// or out of range). It never reports an error: Zalo puts the same id on the
// wire as a number in some frames and as a string in others, and one
// unusable id must degrade to "unknown" (0) instead of failing the whole
// frame or catch-up page it arrived in.
//
// Use it only for id-like and timestamp-like fields. Enums and flags stay
// strictly typed so a genuinely different payload still surfaces as an error.
func flexInt64(b json.RawMessage) int64 {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return 0
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return 0
		}
		b = []byte(strings.TrimSpace(s))
	}
	n, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// flexInt is flexInt64 narrowed to int (see flexInt64).
func flexInt(b json.RawMessage) int {
	return int(flexInt64(b))
}

// flexIDString reads a JSON string as-is, or a JSON integer (of any size) as
// its decimal string, returning "" for anything else (absent, null, bool,
// fractional, object...). It never reports an error (see flexInt64).
func flexIDString(b json.RawMessage) string {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return ""
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return ""
		}
		return s
	}
	n, ok := new(big.Int).SetString(string(b), 10)
	if !ok {
		return ""
	}
	return n.String()
}

// FlexIDString is flexIDString for other packages of this module (the api
// package's response types): a JSON string as-is, a JSON integer of any size
// as its decimal string, "" for anything else. It never reports an error.
func FlexIDString(b json.RawMessage) string { return flexIDString(b) }

// FlexInt is flexInt for other packages of this module (see FlexIDString).
func FlexInt(b json.RawMessage) int { return flexInt(b) }
