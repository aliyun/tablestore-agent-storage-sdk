package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

func TestReleaseBinaryFormatsE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputFile})
	sc := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: fullScope("formats")}
	sc.Scope.RunID = ""
	formats := map[string]string{"pdf": "application/pdf", "doc": "application/msword", "docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "xls": "application/vnd.ms-excel", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "ppt": "application/vnd.ms-powerpoint", "pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation", "jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png"}
	for _, ext := range []string{"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "jpg", "jpeg", "png"} {
		t.Run(ext, func(t *testing.T) {
			data, err := os.ReadFile("testdata/formats/source." + ext)
			if err != nil {
				t.Fatal(err)
			}
			enc := base64.StdEncoding.EncodeToString(data)
			hash := sha256.Sum256(data)
			path := "/.wiki/raw/source." + ext
			item, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: sc, Path: path, ContentBase64: &enc, MediaType: formats[ext], View: sdk.ItemViewFull})
			if err != nil {
				t.Fatal(err)
			}
			if item.ContentBase64 == nil || *item.ContentBase64 != enc || item.ContentSHA256 != hex.EncodeToString(hash[:]) || item.ContentSizeBytes != int64(len(data)) || item.Content != nil {
				t.Fatal("binary write altered content/hash/size")
			}
			got, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, View: sdk.ItemViewFull})
			if err != nil || got.ContentBase64 == nil || *got.ContentBase64 != enc || got.MediaType != formats[ext] {
				t.Fatalf("binary readback: %v", err)
			}
			var raw struct {
				Items     []map[string]any `json:"items"`
				NextToken string           `json:"nextToken"`
			}
			err = c.Do(ctx, "ListItems", &sdk.ListItemsRequest{ItemScopedRequest: sdk.ItemScopedRequest{Type: sdk.ItemTypeMemoryFile, MemoryStoreName: store, Scope: sc.Scope}, PathPrefix: "/.wiki/raw/", View: sdk.ItemViewFull, Limit: 20}, &raw)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, entry := range raw.Items {
				if entry["itemId"] != item.ItemID {
					continue
				}
				found = true
				if _, ok := entry["contentBase64"]; ok {
					t.Fatal("list leaked binary body")
				}
				if _, ok := entry["content"]; ok {
					t.Fatal("list leaked text body")
				}
				if entry["mediaType"] != formats[ext] {
					t.Fatal("list lost mediaType")
				}
			}
			if !found {
				t.Fatal("binary missing from list")
			}
			// Invalid replacement must leave the current item and its version unchanged.
			_, err = c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, ContentBase64: sdk.String(base64.StdEncoding.EncodeToString([]byte("invalid signature"))), MediaType: formats[ext]})
			expectItemValidation(t, err)
			same, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, View: sdk.ItemViewFull})
			if err != nil || same.LatestVersionID != item.LatestVersionID || same.ContentBase64 == nil || *same.ContentBase64 != enc {
				t.Fatalf("invalid mutation changed binary: %v", err)
			}
			old, err := c.GetItemVersion(ctx, &sdk.GetItemVersionRequest{ItemScopedRequest: sc, VersionID: item.LatestVersionID, View: sdk.ItemViewFull})
			if err != nil || old.ContentBase64 == nil || *old.ContentBase64 != enc {
				t.Fatalf("binary version: %v", err)
			}
		})
	}
	t.Run("extension_media_mismatch", func(t *testing.T) {
		data, err := os.ReadFile("testdata/formats/source.pdf")
		if err != nil {
			t.Fatal(err)
		}
		enc := base64.StdEncoding.EncodeToString(data)
		for _, tc := range []struct{ path, media string }{{"/.wiki/raw/wrong.png", "application/pdf"}, {"/.wiki/raw/wrong.pdf", "image/png"}, {"/.wiki/raw/wrong.png", "image/png"}} {
			_, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: sc, Path: tc.path, ContentBase64: &enc, MediaType: tc.media})
			expectItemValidation(t, err)
			_, err = c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, Path: tc.path})
			if status, _ := serviceStatus(err); status != 404 {
				t.Fatalf("invalid file persisted: %v", err)
			}
		}
	})
	t.Run("binary_decoded_size_boundary", func(t *testing.T) {
		// A valid PDF may have trailing whitespace. Pad to the documented decimal
		// 2,000,000-byte limit before Base64 encoding, then add exactly one byte.
		data, err := os.ReadFile("testdata/formats/source.pdf")
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, bytes.Repeat([]byte(" "), 2_000_000-len(data))...)
		enc := base64.StdEncoding.EncodeToString(data)
		item, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: sc, Path: "/.wiki/raw/boundary.pdf", ContentBase64: &enc, MediaType: "application/pdf"})
		if err != nil {
			t.Fatal(err)
		}
		if item.ContentSizeBytes != 2_000_000 {
			t.Fatalf("wrong decoded size: %d", item.ContentSizeBytes)
		}
		tooBig := base64.StdEncoding.EncodeToString(append(data, ' '))
		_, err = c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, ContentBase64: &tooBig, MediaType: "application/pdf"})
		var se *sdk.ServiceError
		if !errors.As(err, &se) || se.StatusCode != 413 || se.Code == "" || se.RequestID == "" {
			t.Fatalf("expected oversized rejection with request ID: %v", err)
		}
		t.Logf("oversize rejected: status=%d code=%s requestId=%s", se.StatusCode, se.Code, se.RequestID)
		got, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, View: sdk.ItemViewFull})
		hash := sha256.Sum256(data)
		if err != nil || got.LatestVersionID != item.LatestVersionID || got.ContentSizeBytes != int64(len(data)) || got.ContentSHA256 != hex.EncodeToString(hash[:]) || got.ContentBase64 == nil || *got.ContentBase64 != enc {
			t.Fatalf("2,000,000-byte source was not preserved byte-for-byte after oversize rejection: %v", err)
		}
	})
}

// Response-loss injection occurs only after the real gateway accepted and
// returned a response. It never retries a mutation or weakens normal responses.
type dropMutationResponse struct {
	base      http.RoundTripper
	mu        sync.Mutex
	remaining int
}

func (d *dropMutationResponse) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := d.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	d.mu.Lock()
	drop := d.remaining > 0 && req.URL.Path == "/AddItem" && resp.StatusCode == 200
	if drop {
		d.remaining--
	}
	d.mu.Unlock()
	if !drop {
		return resp, nil
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return nil, fmt.Errorf("synthetic accepted-response loss")
}

func TestReleaseFileResilienceE2E(t *testing.T) {
	c := extractionClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputFile, sdk.StoreOutputStructured})
	sc := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: fullScope("resilience")}
	t.Run("ambiguous_create_reconciles_without_duplicate_write", func(t *testing.T) {
		transport := &dropMutationResponse{base: http.DefaultTransport, remaining: 1}
		fault := e2eClientWithHTTPClient(t, &http.Client{Transport: transport, Timeout: 60 * time.Second})
		_, err := fault.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: sc, Path: "/response-loss.md", Content: "响应丢失测试项目的暗号是雪豹六号。"})
		var ce *sdk.ClientError
		if !errors.As(err, &ce) {
			t.Fatalf("expected client transport error: %v", err)
		}
		item, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, Path: "/response-loss.md"})
		if err != nil {
			t.Fatal(err)
		}
		tasks := poll(t, 90*time.Second, 2*time.Second, "accepted conversion task becomes visible", func() ([]sdk.MemoryTask, error) { return extractionTasks(t, c, sc), nil }, func(tasks []sdk.MemoryTask) bool { return len(tasks) > 0 })
		if len(tasks) != 1 {
			t.Fatalf("accepted write should submit once: %d", len(tasks))
		}
		waitExtractionTask(t, c, sc, &extractionTrace{requestID: tasks[0].RequestID})
		waitDerived(t, c, sc, item, "雪豹六号", true)
		if tasks := extractionTasks(t, c, sc); len(tasks) != 1 {
			t.Fatalf("accepted write produced duplicate tasks: %d", len(tasks))
		}
		versions, err := c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: sc, ItemID: item.ItemID})
		if err != nil || len(versions.Versions) != 1 {
			t.Fatalf("ambiguous request retried: %+v %v", versions, err)
		}
	})
	t.Run("concurrent_version_guard_has_one_winner", func(t *testing.T) {
		item := addExtractionFile(t, c, sc, "/concurrent.md", "并发项目的暗号是银杉零号。")
		start := make(chan struct{})
		type result struct {
			item  *sdk.Item
			err   error
			trace *extractionTrace
		}
		results := make(chan result, 2)
		for _, fact := range []string{"并发项目的暗号是银杉一号。", "并发项目的暗号是银杉二号。"} {
			fact := fact
			go func() {
				<-start
				mctx, trace := extractionContext()
				r, err := c.UpdateItem(mctx, &sdk.UpdateItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, Content: &fact, Precondition: &sdk.ItemPrecondition{ExpectedVersionID: item.LatestVersionID}, View: sdk.ItemViewFull})
				results <- result{r, err, trace}
			}()
		}
		close(start)
		a, b := <-results, <-results
		if a.err != nil {
			a, b = b, a
		}
		if a.err != nil {
			t.Fatalf("no CAS winner: %v %v", a.err, b.err)
		}
		if status, _ := serviceStatus(b.err); status != 409 {
			t.Fatalf("loser must be conflict: %v", b.err)
		}
		waitExtractionTask(t, c, sc, a.trace)
		waitDerived(t, c, sc, a.item, strings.TrimSuffix(strings.TrimPrefix(*a.item.Content, "并发项目的暗号是"), "。"), true)
		versions, err := c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: sc, ItemID: item.ItemID})
		if err != nil || len(versions.Versions) != 2 {
			t.Fatalf("CAS appended extra history: %+v %v", versions, err)
		}
	})
	t.Run("update_then_delete_does_not_resurrect", func(t *testing.T) {
		item := addExtractionFile(t, c, sc, "/delete-race.md", "删除竞态项目的暗号是苍鹰四号。")
		uctx, utrace := extractionContext()
		updated, err := c.UpdateItem(uctx, &sdk.UpdateItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, Content: sdk.String("删除竞态项目的暗号是苍鹰五号。")})
		if err != nil {
			t.Fatal(err)
		}
		dctx, dtrace := extractionContext()
		_, err = c.DeleteItem(dctx, &sdk.DeleteItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, Precondition: &sdk.ItemPrecondition{ExpectedVersionID: updated.LatestVersionID}})
		if err != nil {
			t.Fatal(err)
		}
		waitExtractionTask(t, c, sc, utrace)
		waitExtractionTask(t, c, sc, dtrace)
		waitNoDerived(t, c, sc, item.ItemID)
		_, err = c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID})
		if status, _ := serviceStatus(err); status != 404 {
			t.Fatalf("deleted source resurrected: %v", err)
		}
	})
	t.Run("redact_latest_cascades_without_changing_current_file", func(t *testing.T) {
		item := addExtractionFile(t, c, sc, "/redact.md", "脱敏项目的暗号是紫鹭八号。")
		derived := waitDerived(t, c, sc, item, "紫鹭八号", true)
		before, err := c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: sc, ItemID: item.ItemID})
		if err != nil {
			t.Fatal(err)
		}
		rctx, trace := extractionContext()
		redacted, err := c.RedactItemVersion(rctx, &sdk.RedactItemVersionRequest{ItemScopedRequest: sc, VersionID: item.LatestVersionID})
		if err != nil || !redacted.Redacted {
			t.Fatalf("redact: %+v %v", redacted, err)
		}
		waitExtractionTask(t, c, sc, trace)
		waitNoDerived(t, c, sc, item.ItemID)
		for _, u := range derived {
			_, err := c.GetMemory(ctx, &sdk.GetMemoryRequest{MemoryStoreName: store, Scope: sc.Scope, MemoryID: u.ID})
			if status, _ := serviceStatus(err); status != 404 {
				t.Fatalf("redacted derived row readable: %v", err)
			}
		}
		got, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID, View: sdk.ItemViewFull})
		if err != nil || got.LatestVersionID != item.LatestVersionID || got.Content == nil || *got.Content != *item.Content {
			t.Fatalf("redaction changed current file: %+v %v", got, err)
		}
		after, err := c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: sc, ItemID: item.ItemID})
		if err != nil || len(after.Versions) != len(before.Versions) {
			t.Fatalf("redact appended history: %v", err)
		}
	})
}

func waitNoDerived(t *testing.T, c *sdk.Client, sc sdk.ItemScopedRequest, id string) {
	t.Helper()
	poll(t, 90*time.Second, 2*time.Second, "derived rows removed", func() ([]sdk.MemoryUnit, error) { return extractionMemories(c, sc) }, func(units []sdk.MemoryUnit) bool {
		for _, u := range units {
			if u.Metadata["source_file_id"] == id {
				return false
			}
		}
		return true
	})
	poll(t, 90*time.Second, 2*time.Second, "derived search removed", func() (*sdk.SearchMemoriesResponse, error) {
		return c.SearchMemories(context.Background(), &sdk.SearchMemoriesRequest{MemoryStoreName: sc.MemoryStoreName, Scope: &sc.Scope, Query: "项目暗号", Metadata: map[string]string{"source_file_id": id}, IncludeEvidence: true, EnableRerank: sdk.Bool(false)})
	}, func(r *sdk.SearchMemoriesResponse) bool { return len(r.Results) == 0 && len(r.Evidence) == 0 })
}
