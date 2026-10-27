package agentstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type fakeUploader struct {
	mu      sync.Mutex
	puts    []string
	deletes []string
	putErr  error
}

func (u *fakeUploader) PutObjectFromFile(_ context.Context, objectKey, _ string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.putErr != nil {
		return u.putErr
	}
	u.puts = append(u.puts, objectKey)
	return nil
}

func (u *fakeUploader) DeleteObjects(_ context.Context, objectKeys []string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.deletes = append(u.deletes, objectKeys...)
	return nil
}

func newUploadTestClient(t *testing.T, uploader *fakeUploader, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(Config{
		Endpoint:      server.URL,
		InstanceName:  "inst",
		APIKey:        "key",
		HTTPClient:    server.Client(),
		OSSBucketName: "bucket",
		OSSUploader:   uploader,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestUploadDocuments(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.pdf")
	if err := os.WriteFile(filePath, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotPayload map[string]any
	uploader := &fakeUploader{}
	client := newUploadTestClient(t, uploader, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/AddDocuments" {
			t.Errorf("action = %q", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		w.Write([]byte(`{"code":"SUCCESS","data":{"documentDetails":[{"docId":"d1","status":"succeed"}]},"message":"succeed","requestId":"r"}`))
	})

	resp, err := client.UploadDocuments(context.Background(), &UploadDocumentsRequest{
		KnowledgeBaseName: "kb",
		Subspace:          "sub",
		Documents: []UploadDocumentItem{
			{FilePath: filePath, Metadata: map[string]any{"author": "aliyun"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(uploader.puts) != 1 || !strings.HasPrefix(uploader.puts[0], "inst/kb/source_doc/sub/") {
		t.Errorf("puts = %v", uploader.puts)
	}
	documents := gotPayload["documents"].([]any)
	document := documents[0].(map[string]any)
	if _, hasFilePath := document["filePath"]; hasFilePath {
		t.Error("filePath must be replaced by ossKey")
	}
	ossKey := document["ossKey"].(string)
	if !strings.HasPrefix(ossKey, "oss://bucket/inst/kb/source_doc/sub/") {
		t.Errorf("ossKey = %q", ossKey)
	}
	if document["metadata"].(map[string]any)["author"] != "aliyun" {
		t.Errorf("metadata = %v", document["metadata"])
	}
	if resp.Data.DocumentDetails[0].DocID != "d1" {
		t.Errorf("resp = %+v", resp)
	}
}

func TestUploadDocumentsFileNotFound(t *testing.T) {
	client := newUploadTestClient(t, &fakeUploader{}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	_, err := client.UploadDocuments(context.Background(), &UploadDocumentsRequest{
		KnowledgeBaseName: "kb",
		Documents:         []UploadDocumentItem{{FilePath: "/nonexistent/file.pdf"}},
	})
	if err == nil || !strings.Contains(err.Error(), "File not found") {
		t.Errorf("err = %v", err)
	}
}

func TestUploadDocumentsPreservesUploaderErrorClassification(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.pdf")
	if err := os.WriteFile(filePath, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}

	cause := errors.New("upload failure marker")
	client := newUploadTestClient(t, &fakeUploader{putErr: cause}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	_, err := client.UploadDocuments(context.Background(), &UploadDocumentsRequest{
		KnowledgeBaseName: "kb",
		Documents:         []UploadDocumentItem{{FilePath: filePath}},
	})

	var clientErr *ClientError
	if !errors.As(err, &clientErr) {
		t.Fatalf("error type = %T, want *ClientError", err)
	}
	if got, want := clientErr.Kind(), ClientErrorKindTransportOther; got != want {
		t.Errorf("ClientError.Kind() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Error("uploader cause was not preserved")
	}
	if got, want := clientErr.Error(), "OTSClientError: Failed to upload file to OSS: upload failure marker"; got != want {
		t.Errorf("ClientError.Error() = %q, want %q", got, want)
	}
}

func TestUploadDocumentsRollbackOnServiceError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.pdf")
	if err := os.WriteFile(filePath, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader := &fakeUploader{}
	client := newUploadTestClient(t, uploader, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code":"INVALID","message":"bad request"}`))
	})
	_, err := client.UploadDocuments(context.Background(), &UploadDocumentsRequest{
		KnowledgeBaseName: "kb",
		Documents:         []UploadDocumentItem{{FilePath: filePath}},
	})
	if _, ok := err.(*ServiceError); !ok {
		t.Fatalf("err = %T %v, want *ServiceError", err, err)
	}
	if len(uploader.deletes) != 1 {
		t.Errorf("uploaded objects not rolled back: %v", uploader.deletes)
	}
}

func TestUploadDocumentsWithoutOSS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(server.Close)
	client, err := NewClient(Config{
		Endpoint:     server.URL,
		InstanceName: "inst",
		APIKey:       "key",
		HTTPClient:   server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.UploadDocuments(context.Background(), &UploadDocumentsRequest{KnowledgeBaseName: "kb"})
	if err == nil || !strings.Contains(err.Error(), "OSS client not initialized") {
		t.Errorf("err = %v", err)
	}
}

func TestDeleteDocumentsWithFile(t *testing.T) {
	uploader := &fakeUploader{}
	client := newUploadTestClient(t, uploader, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/GetDocument":
			w.Write([]byte(`{"code":"SUCCESS","data":[{"docId":"d1","ossKey":"oss://bucket/path/doc1.pdf"}],"message":"succeed","requestId":"r"}`))
		case "/DeleteDocuments":
			w.Write([]byte(`{"code":"SUCCESS","data":{"documentDetails":[]},"message":"succeed","requestId":"r"}`))
		default:
			t.Errorf("unexpected action %q", r.URL.Path)
		}
	})

	_, err := client.DeleteDocumentsWithOptions(context.Background(), &DeleteDocumentsRequest{
		KnowledgeBaseName: "kb",
		Documents: []DeleteDocumentItem{
			{DocID: "d1"},                               // resolved via GetDocument
			{OssKey: "oss://bucket/path/doc2.pdf"},      // same bucket: deleted
			{OssKey: "oss://otherbucket/path/doc3.pdf"}, // different bucket: skipped
		},
	}, DeleteDocumentsOptions{DeleteFile: true})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"path/doc1.pdf": true, "path/doc2.pdf": true}
	if len(uploader.deletes) != 2 {
		t.Fatalf("deletes = %v, want 2 keys", uploader.deletes)
	}
	for _, key := range uploader.deletes {
		if !want[key] {
			t.Errorf("unexpected deleted key %q", key)
		}
	}
}

func TestDeleteDocumentsWithoutFileSkipsOSS(t *testing.T) {
	uploader := &fakeUploader{}
	client := newUploadTestClient(t, uploader, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/DeleteDocuments" {
			t.Errorf("unexpected action %q", r.URL.Path)
		}
		w.Write([]byte(`{"code":"SUCCESS","data":{"documentDetails":[]},"message":"succeed","requestId":"r"}`))
	})
	_, err := client.DeleteDocuments(context.Background(), &DeleteDocumentsRequest{
		KnowledgeBaseName: "kb",
		Documents:         []DeleteDocumentItem{{OssKey: "oss://bucket/path/doc.pdf"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(uploader.deletes) != 0 {
		t.Errorf("deletes = %v, want none", uploader.deletes)
	}
}

func TestDoEscapeHatch(t *testing.T) {
	client := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Heartbeat" {
			t.Errorf("action = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"ok":true}`)
	})
	var result map[string]any
	if err := client.Do(context.Background(), "Heartbeat", map[string]any{}, &result); err != nil {
		t.Fatal(err)
	}
	if result["ok"] != true {
		t.Errorf("result = %v", result)
	}
}
