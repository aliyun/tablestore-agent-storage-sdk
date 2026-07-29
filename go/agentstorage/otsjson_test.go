package agentstorage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newAPIKeyTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(Config{
		Endpoint:     server.URL,
		InstanceName: "instance",
		APIKey:       "secret-key",
		HTTPClient:   server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestAPIKeyRequestHeadersAndBody(t *testing.T) {
	var gotHeaders http.Header
	var gotBody []byte
	var gotPath string
	client := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		gotBody, _ = io.ReadAll(r.Body)
		gotPath = r.URL.Path
		w.Write([]byte(`{"memoryStoreName":"store","status":"running","acceptedMessages":1}`))
	})

	resp, err := client.AddMemories(context.Background(), &AddMemoriesRequest{
		MemoryStoreName: "store",
		Scope:           &Scope{AppID: "app"},
		Text:            "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/AddMemories" {
		t.Errorf("path = %q", gotPath)
	}
	if got := gotHeaders.Get("x-ots-apikey"); got != "secret-key" {
		t.Errorf("x-ots-apikey = %q", got)
	}
	if got := gotHeaders.Get("x-ots-instancename"); got != "instance" {
		t.Errorf("x-ots-instancename = %q", got)
	}
	if got := gotHeaders.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := gotHeaders.Get("User-Agent"); !strings.HasPrefix(got, "tablestore-agent-storage-go/") {
		t.Errorf("User-Agent = %q", got)
	}
	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["text"] != "hello" {
		t.Errorf("body = %s", gotBody)
	}
	if resp.Status != "running" || resp.AcceptedMessages != 1 {
		t.Errorf("resp = %+v", resp)
	}
}

func TestAPIKeyRequiresHTTPS(t *testing.T) {
	_, err := NewClient(Config{
		Endpoint:     "http://example.com",
		InstanceName: "instance",
		APIKey:       "key",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Errorf("err = %v, want HTTPS requirement", err)
	}
}

func TestServiceErrorMapping(t *testing.T) {
	client := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-ots-requestid", "req-1")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"code":"CONFLICT","message":"ingest task index is still building","requestId":"req-1"}`))
	})
	_, err := client.GetMemoryTask(context.Background(), &GetMemoryTaskRequest{
		MemoryStoreName: "store",
		RequestID:       "rid",
	})
	serviceErr, ok := err.(*ServiceError)
	if !ok {
		t.Fatalf("err = %T %v, want *ServiceError", err, err)
	}
	if serviceErr.StatusCode != 409 || serviceErr.Code != "CONFLICT" || serviceErr.RequestID != "req-1" {
		t.Errorf("serviceErr = %+v", serviceErr)
	}
}

// signedTestServer answers with fully valid signed-mode response headers.
func signedTestServer(t *testing.T, accessKeyID, accessKeySecret string, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := []byte(body)
		responseHeaders := map[string]string{
			"x-ots-contentmd5":  md5Base64(payload),
			"x-ots-requestid":   "req-1",
			"x-ots-date":        buildDateHeader(time.Now()),
			"x-ots-contenttype": "application/json",
		}
		for key, value := range responseHeaders {
			w.Header().Set(key, value)
		}
		signature := responseSignature(r.URL.Path, responseHeaders, []byte(accessKeySecret))
		w.Header().Set("Authorization", "OTS "+accessKeyID+":"+signature)
		w.Write(payload)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestSignedRequestRoundTrip(t *testing.T) {
	server := signedTestServer(t, "ak", "sk", `{"code":"SUCCESS","data":{},"message":"succeed","requestId":"req-1"}`)
	client, err := NewClient(Config{
		Endpoint:        server.URL,
		InstanceName:    "instance",
		AccessKeyID:     "ak",
		AccessKeySecret: "sk",
		STSToken:        "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.CreateKnowledgeBase(context.Background(), &CreateKnowledgeBaseRequest{
		KnowledgeBaseName: "kb",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code != "SUCCESS" || resp.RequestID != "req-1" {
		t.Errorf("resp = %+v", resp)
	}
}

func TestSignedRequestSendsSignatureHeaders(t *testing.T) {
	var gotHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		// Missing response headers: client must reject the response.
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(Config{
		Endpoint:        server.URL,
		InstanceName:    "instance",
		AccessKeyID:     "ak",
		AccessKeySecret: "sk",
		STSToken:        "token",
		ExtraHeaders:    map[string]string{"x-ots-extra": "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateKnowledgeBase(context.Background(), &CreateKnowledgeBaseRequest{KnowledgeBaseName: "kb"})
	if err == nil || !strings.Contains(err.Error(), "missing in response header") {
		t.Errorf("err = %v, want missing-header validation error", err)
	}
	for _, header := range []string{"x-ots-date", "x-ots-apiversion", "x-ots-accesskeyid", "x-ots-instancename", "x-ots-contentmd5", "x-ots-signature", "x-ots-ststoken", "x-ots-extra"} {
		if gotHeaders.Get(header) == "" {
			t.Errorf("request header %q missing", header)
		}
	}
}

func TestSignedResponseMD5Mismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-ots-contentmd5", "bogus")
		w.Header().Set("x-ots-requestid", "req-1")
		w.Header().Set("x-ots-date", buildDateHeader(time.Now()))
		w.Header().Set("x-ots-contenttype", "application/json")
		w.Header().Set("Authorization", "OTS ak:sig")
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(Config{
		Endpoint:        server.URL,
		InstanceName:    "instance",
		AccessKeyID:     "ak",
		AccessKeySecret: "sk",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateKnowledgeBase(context.Background(), &CreateKnowledgeBaseRequest{KnowledgeBaseName: "kb"})
	if err == nil || !strings.Contains(err.Error(), "MD5 mismatch") {
		t.Errorf("err = %v, want MD5 mismatch", err)
	}
}

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(Config{InstanceName: "i", APIKey: "k"}); err == nil {
		t.Error("missing endpoint should fail")
	}
	if _, err := NewClient(Config{Endpoint: "https://e", InstanceName: "i"}); err == nil {
		t.Error("missing credentials should fail")
	}
	if _, err := NewClient(Config{Endpoint: "https://e", InstanceName: "i", AccessKeyID: "a", AccessKeySecret: "s", SignVersion: SignV4}); err == nil {
		t.Error("v4 without region should fail")
	}
	if _, err := NewClient(Config{Endpoint: "http://e", InstanceName: "i", APIKey: "k"}); err == nil {
		t.Error("api key over http should fail by default")
	}
	if _, err := NewClient(Config{Endpoint: "http://e", InstanceName: "i", APIKey: "k", AllowInsecureAPIKey: true}); err != nil {
		t.Errorf("api key over http with AllowInsecureAPIKey should pass, got %v", err)
	}
}
