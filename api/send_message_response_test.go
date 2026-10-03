package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/amrakk/zcago/errs"
	"github.com/amrakk/zcago/internal/cryptox"
	"github.com/amrakk/zcago/model"
	"github.com/amrakk/zcago/session"
)

// The regression these tests exist for: SendMessageResult.MsgID was a plain
// `string`, zca-js types the send response's msgId as a number, and a JSON
// number failed the decode -- which httpx turns into
// "ZaloAPIError[0]: Failed to parse response data", so a message Zalo had
// already DELIVERED was reported to the caller as failed.

func TestSendMessageResultMsgIDShapes(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		{"number", `{"msgId":7123456789012}`, "7123456789012"},
		{"string", `{"msgId":"7123456789012"}`, "7123456789012"},
		{"number beyond float64 precision", `{"msgId":9007199254740993}`, "9007199254740993"},
		{"number beyond int64", `{"msgId":92233720368547758070}`, "92233720368547758070"},
		{"absent", `{}`, ""},
		{"null", `{"msgId":null}`, ""},
		{"empty string", `{"msgId":""}`, ""},
		{"bool", `{"msgId":true}`, ""},
		{"object", `{"msgId":{"id":1}}`, ""},
		{"array", `{"msgId":[1]}`, ""},
		{"fractional", `{"msgId":1.5}`, ""},
		{"unknown extra fields", `{"msgId":42,"status":"ok","n":[1,2]}`, "42"},
		{"payload is a bare number", `42`, ""},
		{"payload is a bare string", `"ok"`, ""},
		{"payload is an array", `[{"msgId":1}]`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r SendMessageResult
			if err := json.Unmarshal([]byte(tc.in), &r); err != nil {
				t.Fatalf("decode must never fail (the message was already accepted), got %v", err)
			}
			if r.MsgID != tc.want {
				t.Fatalf("MsgID = %q, want %q", r.MsgID, tc.want)
			}
		})
	}
}

// A number and a quoted string must yield the SAME id: the bridge stores it
// as the message's primary key and matches the listener echo against it.
func TestSendMessageResultNumberAndStringAgree(t *testing.T) {
	var num, str SendMessageResult
	if err := json.Unmarshal([]byte(`{"msgId":6900000000001}`), &num); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"msgId":"6900000000001"}`), &str); err != nil {
		t.Fatal(err)
	}
	if num.MsgID != str.MsgID || num.MsgID != "6900000000001" {
		t.Fatalf("number -> %q, string -> %q, want both 6900000000001", num.MsgID, str.MsgID)
	}
}

// CliMsgID is library-side bookkeeping, never read from (or written to) the
// wire.
func TestSendMessageResultCliMsgIDIsNotOnTheWire(t *testing.T) {
	r := SendMessageResult{CliMsgID: "keep"}
	if err := json.Unmarshal([]byte(`{"msgId":1,"cliMsgId":"999","CliMsgID":"999"}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.CliMsgID != "keep" {
		t.Fatalf("CliMsgID = %q, the response must not set it", r.CliMsgID)
	}
	out, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "keep") {
		t.Fatalf("CliMsgID leaked into JSON: %s", out)
	}
}

// --- the whole send path, against a fake transport (no network) ---

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

var testSecretKey = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

// capturedSend is one request the fake transport saw: its path and the
// decrypted `params` it carried.
type capturedSend struct {
	path   string
	params map[string]any
}

// newSendHarness builds a real SendMessage endpoint over a sealed session
// whose HTTP client answers every request with `data` as the encrypted
// payload's data field (errorCode as its error_code).
func newSendHarness(t *testing.T, innerJSON string) (SendMessageFn, *[]capturedSend) {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(testSecretKey)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var seen []capturedSend
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		got := capturedSend{path: r.URL.Path}
		if enc := form.Get("params"); enc != "" {
			plain, derr := cryptox.DecodeAESCBC(key, enc)
			if derr != nil {
				t.Errorf("request params did not decrypt: %v", derr)
			} else if uerr := json.Unmarshal(plain, &got.params); uerr != nil {
				t.Errorf("request params are not JSON: %v", uerr)
			}
		}
		mu.Lock()
		seen = append(seen, got)
		mu.Unlock()

		enc, eerr := cryptox.EncodeAESCBC(key, innerJSON, cryptox.EncryptTypeBase64)
		if eerr != nil {
			t.Errorf("encrypt response: %v", eerr)
		}
		outer, _ := json.Marshal(map[string]any{"error_code": 0, "error_message": "", "data": enc})
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(string(outer))),
			Request:    r,
		}, nil
	})}

	sc := session.NewContext(session.WithHTTPClient(client))
	info := &session.LoginInfo{UID: "1000"}
	info.ZpwServiceMapV3.Chat = []string{"https://chat.zalo.test"}
	info.ZpwServiceMapV3.Group = []string{"https://group.zalo.test"}
	info.ZpwServiceMapV3.File = []string{"https://file.zalo.test"}
	settings := &session.Settings{}
	settings.Features.ShareFile.MaxFile = 10
	sc.SealLogin(session.Seal{
		UID:       "1000",
		IMEI:      "test-imei",
		UserAgent: "test-agent",
		SecretKey: session.SecretKey(testSecretKey),
		LoginInfo: info,
		Settings:  settings,
	})
	a := &api{sc: sc}
	fn, err := sendMessageFactory(sc, a)
	if err != nil {
		t.Fatalf("sendMessageFactory: %v", err)
	}
	return fn, &seen
}

func TestSendMessageTextDecodesIDAndReturnsClientID(t *testing.T) {
	for _, tc := range []struct {
		name, inner, wantID string
		threadType          model.ThreadType
		wantPath            string
	}{
		{"user, numeric id", `{"error_code":0,"data":{"msgId":7300000000001}}`, "7300000000001", model.ThreadTypeUser, "/api/message/sms"},
		{"user, string id", `{"error_code":0,"data":{"msgId":"7300000000001"}}`, "7300000000001", model.ThreadTypeUser, "/api/message/sms"},
		{"group, numeric id", `{"error_code":0,"data":{"msgId":7300000000002}}`, "7300000000002", model.ThreadTypeGroup, "/api/group/sendmsg"},
		{"group, string id", `{"error_code":0,"data":{"msgId":"7300000000002"}}`, "7300000000002", model.ThreadTypeGroup, "/api/group/sendmsg"},
		{"id absent", `{"error_code":0,"data":{}}`, "", model.ThreadTypeUser, "/api/message/sms"},
		{"id null", `{"error_code":0,"data":{"msgId":null}}`, "", model.ThreadTypeUser, "/api/message/sms"},
		{"id garbage", `{"error_code":0,"data":{"msgId":{"x":1}}}`, "", model.ThreadTypeUser, "/api/message/sms"},
		{"data is not an object", `{"error_code":0,"data":"ok"}`, "", model.ThreadTypeUser, "/api/message/sms"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			send, seen := newSendHarness(t, tc.inner)
			resp, err := send(context.Background(), "2000", tc.threadType, MessageContent{Msg: "hello"})
			if err != nil {
				t.Fatalf("a delivered message must not be reported as failed, got %v", err)
			}
			if resp == nil || resp.Message == nil {
				t.Fatalf("expected a message result, got %+v", resp)
			}
			if resp.Message.MsgID != tc.wantID {
				t.Fatalf("MsgID = %q, want %q", resp.Message.MsgID, tc.wantID)
			}
			if len(*seen) != 1 {
				t.Fatalf("expected exactly one request, saw %d", len(*seen))
			}
			req := (*seen)[0]
			if req.path != tc.wantPath {
				t.Fatalf("request path = %q, want %q (endpoints must not change)", req.path, tc.wantPath)
			}
			wire, ok := req.params["clientId"].(float64)
			if !ok {
				t.Fatalf("wire clientId is %T, want a JSON number as before", req.params["clientId"])
			}
			if resp.Message.CliMsgID == "" {
				t.Fatal("CliMsgID must carry the clientId that went on the wire")
			}
			if got := json.Number(resp.Message.CliMsgID).String(); got != jsonInt(wire) {
				t.Fatalf("CliMsgID = %q, wire clientId = %s", got, jsonInt(wire))
			}
		})
	}
}

func jsonInt(f float64) string {
	b, _ := json.Marshal(int64(f))
	return string(b)
}

// An application-level error must still surface as a ZaloAPIError carrying
// Zalo's code: the lenient id decode must not swallow real failures.
func TestSendMessageStillReportsZaloErrors(t *testing.T) {
	send, _ := newSendHarness(t, `{"error_code":221,"error_message":"rate limited","data":null}`)
	_, err := send(context.Background(), "2000", model.ThreadTypeUser, MessageContent{Msg: "hello"})
	var ze errs.ZaloAPIError
	if !asZaloAPIError(err, &ze) || ze.Code == nil || *ze.Code != 221 {
		t.Fatalf("expected ZaloAPIError[221], got %v", err)
	}
}

func asZaloAPIError(err error, target *errs.ZaloAPIError) bool {
	ze, ok := err.(errs.ZaloAPIError)
	if ok {
		*target = ze
	}
	return ok
}

// The image send (photo_original/send) carries its clientId as a string on
// the wire; the payload's ClientID must be that same value so the result
// can report it.
func TestPrepareAttachmentPayloadsExposeClientID(t *testing.T) {
	uploads := UploadAttachmentResponse{
		{FileType: model.FileTypeImage, TotalSize: 10, Image: &UploadImageInfo{PhotoID: "55", NormalURL: "n", HDURL: "h", ThumbURL: "t"}},
		{FileType: model.FileTypeImage, TotalSize: 10, Image: &UploadImageInfo{PhotoID: "56", NormalURL: "n", HDURL: "h", ThumbURL: "t"}},
		{FileType: model.FileTypeOther, TotalSize: 10, ClientFileID: 4242, File: &UploadFileInfo{FileID: "9", FileName: "a.pdf"}},
	}
	payloads, err := prepareAttachmentPayloads(uploads, "2000", model.ThreadTypeUser, MessageContent{}, nil, false, "layout", 1700000000000)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1700000000000", "1700000000001", "4242"}
	for i, p := range payloads {
		if p.ClientID != want[i] {
			t.Fatalf("payload %d ClientID = %q, want %q", i, p.ClientID, want[i])
		}
	}
	for i := 0; i < 2; i++ {
		if wire, _ := payloads[i].Params["clientId"].(string); wire != want[i] {
			t.Fatalf("payload %d wire clientId = %v, want %q (wire format must not change)", i, payloads[i].Params["clientId"], want[i])
		}
	}
	if wire, _ := payloads[2].Params["clientId"].(int); wire != 4242 {
		t.Fatalf("file payload wire clientId = %v, want the int 4242 as before", payloads[2].Params["clientId"])
	}
}

func TestUploadRawResponseLenientIDs(t *testing.T) {
	for _, tc := range []struct {
		name, in  string
		wantPhoto string
		wantNil   bool
		wantCFID  int
		wantChunk int
		wantDone  bool
	}{
		{"group shape: numeric photoId", `{"finished":1,"clientFileId":7,"chunkId":1,"photoId":9007199254740993}`, "9007199254740993", false, 7, 1, true},
		{"user shape: string photoId", `{"finished":true,"clientFileId":7,"chunkId":1,"photoId":"9007199254740993"}`, "9007199254740993", false, 7, 1, true},
		{"quoted bookkeeping ids", `{"finished":0,"clientFileId":"7","chunkId":"2","photoId":"-1"}`, "-1", false, 7, 2, false},
		{"photoId absent", `{"finished":1,"clientFileId":7,"chunkId":1}`, "", true, 7, 1, true},
		{"photoId null", `{"photoId":null}`, "", true, 0, 0, false},
		{"photoId garbage", `{"photoId":{"a":1},"clientFileId":[1],"chunkId":true}`, "", true, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r rawResponse
			if err := json.Unmarshal([]byte(tc.in), &r); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil != (r.PhotoID == nil) {
				t.Fatalf("PhotoID nil = %v, want %v", r.PhotoID == nil, tc.wantNil)
			}
			if r.PhotoID != nil && *r.PhotoID != tc.wantPhoto {
				t.Fatalf("PhotoID = %q, want %q", *r.PhotoID, tc.wantPhoto)
			}
			if r.ClientFileID != tc.wantCFID || r.ChunkID != tc.wantChunk || r.Finished != tc.wantDone {
				t.Fatalf("got clientFileId=%d chunkId=%d finished=%v, want %d %d %v",
					r.ClientFileID, r.ChunkID, r.Finished, tc.wantCFID, tc.wantChunk, tc.wantDone)
			}
		})
	}
}
