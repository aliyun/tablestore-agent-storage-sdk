// Package e2e verifies the Go SDK against a live memory service. The tests are
// skipped unless OTS_ENDPOINT / OTS_INSTANCE_NAME plus a credential
// (OTS_ACCESS_KEY_ID + OTS_ACCESS_KEY_SECRET, or OTS_API_KEY) are set:
//
//	OTS_ENDPOINT=http://host OTS_INSTANCE_NAME=inst \
//	OTS_ACCESS_KEY_ID=... OTS_ACCESS_KEY_SECRET=... \
//	go test -v -timeout 20m ./e2e
package e2e

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

func e2eClient(t *testing.T) *agentstorage.Client {
	t.Helper()
	endpoint := os.Getenv("OTS_ENDPOINT")
	instance := os.Getenv("OTS_INSTANCE_NAME")
	accessKeyID := os.Getenv("OTS_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("OTS_ACCESS_KEY_SECRET")
	// OTS_API_KEY is the alternative credential (as in the TypeScript apikey
	// e2e suite). Over a plain-http endpoint it needs the insecure opt-in,
	// which only makes sense against a locally running service.
	apiKey := os.Getenv("OTS_API_KEY")
	hasAccessKeys := accessKeyID != "" && accessKeySecret != ""
	if endpoint == "" || instance == "" || (!hasAccessKeys && apiKey == "") {
		t.Skip("e2e env not configured (OTS_ENDPOINT/OTS_INSTANCE_NAME plus OTS_ACCESS_KEY_ID/OTS_ACCESS_KEY_SECRET or OTS_API_KEY)")
	}
	client, err := agentstorage.NewClient(agentstorage.Config{
		Endpoint:            endpoint,
		InstanceName:        instance,
		AccessKeyID:         accessKeyID,
		AccessKeySecret:     accessKeySecret,
		APIKey:              apiKey,
		AllowInsecureAPIKey: apiKey != "" && strings.HasPrefix(endpoint, "http://"),
		Region:              os.Getenv("OTS_REGION"),
		SignVersion:         agentstorage.SignVersion(os.Getenv("OTS_SIGN_VERSION")),
		Timeout:             60 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

func suffix(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func fullScope(label string) agentstorage.Scope {
	return agentstorage.Scope{
		AppID:    fmt.Sprintf("go_app_%s_%s", label, suffix(8)),
		TenantID: fmt.Sprintf("tenant_%s_%s", label, suffix(4)),
		AgentID:  fmt.Sprintf("agent_%s_%s", label, suffix(4)),
		RunID:    fmt.Sprintf("go_run_%s_%s", label, suffix(8)),
	}
}

func serviceStatus(err error) (int, string) {
	var serviceErr *agentstorage.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.StatusCode, serviceErr.Code
	}
	return 0, ""
}

func isIndexBuilding(err error) bool {
	status, code := serviceStatus(err)
	// The conflict code differs by deployment ("OTSOperationConflict" from
	// Tablestore itself, "CONFLICT" from the memory service), so the message is
	// the reliable signal.
	return status == 409 && (code == "OTSOperationConflict" || code == "CONFLICT") &&
		strings.Contains(strings.ToLower(err.Error()), "secondary index is still building")
}

// poll runs action every interval until predicate passes or the deadline hits.
func poll[T any](t *testing.T, timeout, interval time.Duration, description string,
	action func() (T, error), predicate func(T) bool) T {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last T
	var lastErr error
	for time.Now().Before(deadline) {
		value, err := action()
		lastErr = err
		if err == nil {
			last = value
			if predicate(value) {
				return value
			}
		}
		time.Sleep(interval)
	}
	t.Fatalf("%s: timed out; last value %+v, last err %v", description, last, lastErr)
	return last
}

// retry runs action up to attempts times, retrying while shouldRetry(err).
func retry[T any](t *testing.T, attempts int, interval time.Duration,
	action func() (T, error), shouldRetry func(error) bool) (T, error) {
	t.Helper()
	var value T
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		value, err = action()
		if err == nil || !shouldRetry(err) {
			return value, err
		}
		time.Sleep(interval)
	}
	return value, err
}

func memoryListContains(memories []agentstorage.MemoryUnit, fragment string) bool {
	for _, memory := range memories {
		if strings.Contains(strings.ToLower(memory.Text), strings.ToLower(fragment)) {
			return true
		}
	}
	return false
}

func waitForListMatch(t *testing.T, ctx context.Context, client *agentstorage.Client,
	store string, scope agentstorage.Scope, fragment, description string) *agentstorage.ListMemoriesResponse {
	t.Helper()
	return poll(t, 90*time.Second, 2*time.Second, description,
		func() (*agentstorage.ListMemoriesResponse, error) {
			return client.ListMemories(ctx, &agentstorage.ListMemoriesRequest{
				MemoryStoreName: store,
				Scope:           &scope,
				Limit:           50,
			})
		},
		func(resp *agentstorage.ListMemoriesResponse) bool {
			return memoryListContains(resp.Memories, fragment)
		})
}

// waitForMemoryIDIndexReady probes GetMemory with a missing id until the
// memory-id index answers 404 instead of 409 (index building).
func waitForMemoryIDIndexReady(t *testing.T, ctx context.Context, client *agentstorage.Client,
	store string, scope agentstorage.Scope) {
	t.Helper()
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		_, err := client.GetMemory(ctx, &agentstorage.GetMemoryRequest{
			MemoryStoreName: store,
			MemoryID:        "warmup-missing-memory-id",
			Scope:           scope,
		})
		if err == nil {
			return
		}
		if status, _ := serviceStatus(err); status == 404 {
			return
		}
		if isIndexBuilding(err) {
			time.Sleep(2 * time.Second)
			continue
		}
		t.Fatalf("memory id index warmup: %v", err)
	}
	t.Fatal("memory id index did not become ready in time")
}

func TestMemoryE2E(t *testing.T) {
	client := e2eClient(t)
	ctx := context.Background()
	store := fmt.Sprintf("gomem_%d_%s", time.Now().Unix(), suffix(6))

	defer func() {
		_, _ = client.DeleteMemoryStore(ctx, &agentstorage.DeleteMemoryStoreRequest{MemoryStoreName: store})
	}()

	// --- Memory store lifecycle ---
	created, err := client.CreateMemoryStore(ctx, &agentstorage.CreateMemoryStoreRequest{
		MemoryStoreName: store,
		Description:     "Go SDK memory e2e",
	})
	if err != nil {
		t.Fatalf("CreateMemoryStore: %v", err)
	}
	if created.MemoryStoreName != store {
		t.Fatalf("CreateMemoryStore name = %q, want %q", created.MemoryStoreName, store)
	}
	t.Logf("created store %s (storageMode=%q createdAt=%s)", store, created.StorageMode, created.CreatedAt)

	fetched, err := client.GetMemoryStore(ctx, &agentstorage.GetMemoryStoreRequest{MemoryStoreName: store})
	if err != nil {
		t.Fatalf("GetMemoryStore: %v", err)
	}
	if fetched.Description != "Go SDK memory e2e" {
		t.Errorf("GetMemoryStore description = %q", fetched.Description)
	}

	stores, err := client.ListMemoryStores(ctx, &agentstorage.ListMemoryStoresRequest{Limit: 100})
	if err != nil {
		t.Fatalf("ListMemoryStores: %v", err)
	}
	found := false
	for _, item := range stores.Stores {
		if item.MemoryStoreName == store {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListMemoryStores does not contain %s", store)
	}

	updatedStore, err := client.UpdateMemoryStore(ctx, &agentstorage.UpdateMemoryStoreRequest{
		MemoryStoreName: store,
		Description:     agentstorage.String("Go SDK memory e2e updated"),
	})
	if err != nil {
		t.Fatalf("UpdateMemoryStore: %v", err)
	}
	if updatedStore.Description != "Go SDK memory e2e updated" {
		t.Errorf("UpdateMemoryStore description = %q", updatedStore.Description)
	}

	// --- Warm up the fresh store (mirrors the TS e2e suite) ---
	t.Log("warming up store (60s)...")
	time.Sleep(60 * time.Second)
	warmScope := fullScope("warm")
	_, err = retry(t, 5, 2*time.Second, func() (*agentstorage.AddMemoriesResponse, error) {
		return client.AddMemories(ctx, &agentstorage.AddMemoriesRequest{
			MemoryStoreName: store,
			Scope:           &warmScope,
			Text:            "Go SDK warm sentinel memory for store readiness.",
			Sync:            true,
		})
	}, func(err error) bool {
		status, _ := serviceStatus(err)
		return status == 500
	})
	if err != nil {
		t.Fatalf("warm AddMemories: %v", err)
	}
	waitForListMatch(t, ctx, client, store, warmScope, "warm sentinel", "warm sentinel not list-visible")
	waitForMemoryIDIndexReady(t, ctx, client, store, warmScope)
	t.Log("store warmed up")

	// --- Conversation messages (short-term memory) ---
	conversationScope := fullScope("messages")
	addMessages, err := retry(t, 5, 2*time.Second, func() (*agentstorage.AddMemoriesResponse, error) {
		return client.AddMemories(ctx, &agentstorage.AddMemoriesRequest{
			MemoryStoreName: store,
			Scope:           &conversationScope,
			Messages: []agentstorage.MessagePayload{
				{Role: "user", Content: "Ian enjoys espresso in the morning.", MessageID: "msg_" + suffix(6) + "_1"},
				{Role: "assistant", Content: "I will remember Ian enjoys espresso.", MessageID: "msg_" + suffix(6) + "_2"},
			},
			Metadata: map[string]string{"source": "go-e2e"},
			Sync:     true,
		})
	}, func(err error) bool {
		status, _ := serviceStatus(err)
		return status == 500
	})
	if err != nil {
		t.Fatalf("AddMemories(messages): %v", err)
	}
	if addMessages.MemoryStoreName != store || addMessages.AcceptedMessages != 2 {
		t.Errorf("AddMemories(messages) = %+v", addMessages)
	}
	t.Logf("added conversation: status=%s accepted=%d requestId=%s",
		addMessages.Status, addMessages.AcceptedMessages, addMessages.RequestID)

	waitForListMatch(t, ctx, client, store, conversationScope, "espresso", "conversation memory not list-visible")

	messages, err := client.ListMemoryStoreMessages(ctx, &agentstorage.ListMemoryStoreMessagesRequest{
		MemoryStoreName: store,
		Scope:           conversationScope,
		Limit:           20,
	})
	if err != nil {
		t.Fatalf("ListMemoryStoreMessages: %v", err)
	}
	if messages.Session.Scope.AppID != conversationScope.AppID {
		t.Errorf("session scope = %+v", messages.Session.Scope)
	}
	if len(messages.Session.Messages) < 2 {
		t.Errorf("session messages = %d, want >= 2", len(messages.Session.Messages))
	}
	hasUserRole := false
	for _, message := range messages.Session.Messages {
		if message.Role == "user" {
			hasUserRole = true
		}
	}
	if !hasUserRole {
		t.Error("no user-role message in session")
	}

	// --- Request audit records ---
	requests, err := client.ListMemoryStoreRequests(ctx, &agentstorage.ListMemoryStoreRequestsRequest{
		MemoryStoreName: store,
		Scope:           &agentstorage.Scope{AppID: conversationScope.AppID},
		Operation:       "AddMemories",
		Limit:           20,
	})
	if err != nil {
		t.Fatalf("ListMemoryStoreRequests: %v", err)
	}
	if requests.MemoryStoreName != store {
		t.Errorf("requests store = %q", requests.MemoryStoreName)
	}
	for _, record := range requests.Requests {
		if record.Operation != "AddMemories" {
			t.Errorf("audit record operation = %q", record.Operation)
		}
	}
	t.Logf("audit records: %d", len(requests.Requests))

	// --- Async ingestion task APIs ---
	if addMessages.RequestID != "" {
		task := poll(t, 120*time.Second, 3*time.Second, "ingest task not completed",
			func() (*agentstorage.GetMemoryTaskResponse, error) {
				resp, err := client.GetMemoryTask(ctx, &agentstorage.GetMemoryTaskRequest{
					MemoryStoreName: store,
					RequestID:       addMessages.RequestID,
				})
				if err != nil {
					if status, _ := serviceStatus(err); status == 409 || status == 404 {
						return nil, nil // index still building; keep polling
					}
					return nil, err
				}
				return resp, nil
			},
			func(resp *agentstorage.GetMemoryTaskResponse) bool {
				return resp != nil && resp.Task.Status == agentstorage.MemoryTaskStatusCompleted
			})
		if task.Task.AcceptedMessages != 2 {
			t.Errorf("task acceptedMessages = %d", task.Task.AcceptedMessages)
		}
		t.Logf("ingest task: status=%s units=%d", task.Task.Status, len(task.Task.DerivedUnitIDs))

		tasks, err := client.ListMemoryTasks(ctx, &agentstorage.ListMemoryTasksRequest{
			MemoryStoreName: store,
			Scope:           conversationScope,
			Limit:           10,
		})
		if err != nil {
			t.Fatalf("ListMemoryTasks: %v", err)
		}
		if len(tasks.Tasks) == 0 {
			t.Error("ListMemoryTasks returned no tasks")
		}
	}

	// Scope listing is served from the async-synced search index, so poll.
	scopes := poll(t, 120*time.Second, 3*time.Second, "scopes did not materialize",
		func() (*agentstorage.ListMemoryStoreScopesResponse, error) {
			return client.ListMemoryStoreScopes(ctx, &agentstorage.ListMemoryStoreScopesRequest{
				MemoryStoreName: store,
				Scope:           agentstorage.Scope{AppID: conversationScope.AppID, TenantID: "*", AgentID: "*", RunID: "*"},
				Limit:           10,
			})
		},
		func(resp *agentstorage.ListMemoryStoreScopesResponse) bool {
			return len(resp.Scopes) > 0
		})
	if scopes.Scopes[0].AppID != conversationScope.AppID {
		t.Errorf("scope appId = %q, want %q", scopes.Scopes[0].AppID, conversationScope.AppID)
	}
	t.Logf("scopes listed: %d", len(scopes.Scopes))

	// --- Single-memory CRUD ---
	crudScope := fullScope("crud")
	addText, err := client.AddMemories(ctx, &agentstorage.AddMemoriesRequest{
		MemoryStoreName: store,
		Scope:           &crudScope,
		Text:            "Victor is a pilot based in Kunming.",
		Metadata:        map[string]string{"city": "kunming", "source": "go-e2e"},
		Sync:            true,
	})
	if err != nil {
		t.Fatalf("AddMemories(text): %v", err)
	}
	if addText.AcceptedMessages != 1 {
		t.Errorf("AddMemories(text) accepted = %d", addText.AcceptedMessages)
	}

	listed := waitForListMatch(t, ctx, client, store, crudScope, "Kunming", "text memory not list-visible")
	memoryID := ""
	for _, memory := range listed.Memories {
		if strings.Contains(memory.Text, "Kunming") {
			memoryID = memory.ID
			break
		}
	}
	if memoryID == "" {
		t.Fatal("memory containing Kunming not found")
	}

	got, err := client.GetMemory(ctx, &agentstorage.GetMemoryRequest{
		MemoryStoreName: store,
		MemoryID:        memoryID,
		Scope:           crudScope,
	})
	if err != nil {
		t.Fatalf("GetMemory: %v", err)
	}
	if !strings.Contains(got.Memory.Text, "Kunming") {
		t.Errorf("GetMemory text = %q", got.Memory.Text)
	}

	updated, err := retry(t, 5, 2*time.Second, func() (*agentstorage.UpdateMemoryResponse, error) {
		return client.UpdateMemory(ctx, &agentstorage.UpdateMemoryRequest{
			MemoryStoreName: store,
			MemoryID:        memoryID,
			Scope:           crudScope,
			Text:            "Victor is a pilot based in Chengdu.",
			Metadata:        map[string]string{"city": "chengdu", "source": "go-e2e-updated"},
		})
	}, func(err error) bool {
		status, code := serviceStatus(err)
		return (status == 404 && code == "OTSObjectNotExist") || isIndexBuilding(err)
	})
	if err != nil {
		t.Fatalf("UpdateMemory: %v", err)
	}
	if !updated.Updated || !strings.Contains(updated.Memory.Text, "Chengdu") {
		t.Errorf("UpdateMemory = %+v", updated)
	}

	search := poll(t, 120*time.Second, 2*time.Second, "updated memory not searchable",
		func() (*agentstorage.SearchMemoriesResponse, error) {
			return client.SearchMemories(ctx, &agentstorage.SearchMemoriesRequest{
				MemoryStoreName: store,
				Scope:           &crudScope,
				Query:           "Victor Chengdu pilot",
				TopK:            5,
				EnableRerank:    agentstorage.Bool(false),
			})
		},
		func(resp *agentstorage.SearchMemoriesResponse) bool {
			for _, hit := range resp.Results {
				if strings.Contains(hit.Unit.Text, "Chengdu") {
					return true
				}
			}
			return false
		})
	if search.MemoryStoreName != store {
		t.Errorf("search store = %q", search.MemoryStoreName)
	}
	t.Logf("search hits: %d (top score %.3f)", len(search.Results), search.Results[0].Score)

	deleted, err := retry(t, 10, 3*time.Second, func() (*agentstorage.DeleteMemoryResponse, error) {
		return client.DeleteMemory(ctx, &agentstorage.DeleteMemoryRequest{
			MemoryStoreName: store,
			MemoryID:        memoryID,
			Scope:           crudScope,
		})
	}, isIndexBuilding)
	if err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}
	if !deleted.Deleted || deleted.MemoryID != memoryID {
		t.Errorf("DeleteMemory = %+v", deleted)
	}

	poll(t, 120*time.Second, 2*time.Second, "deleted memory still readable",
		func() (bool, error) {
			resp, err := client.GetMemory(ctx, &agentstorage.GetMemoryRequest{
				MemoryStoreName: store,
				MemoryID:        memoryID,
				Scope:           crudScope,
			})
			if err != nil {
				if status, _ := serviceStatus(err); status == 404 {
					return true, nil
				}
				if isIndexBuilding(err) {
					return false, nil
				}
				return false, err
			}
			return resp.Memory.Deleted, nil
		},
		func(gone bool) bool { return gone })

	// --- Dream APIs (best-effort: depends on server-side LLM configuration) ---
	dream, err := client.CreateMemoryDreamTask(ctx, &agentstorage.CreateMemoryDreamTaskRequest{
		MemoryStoreName: store,
		Scopes:          []agentstorage.Scope{{AppID: conversationScope.AppID, TenantID: conversationScope.TenantID}},
		ApplyMode:       agentstorage.DreamApplyModeProposal,
	})
	if err != nil {
		t.Logf("CreateMemoryDreamTask unavailable on this server, skipping dream checks: %v", err)
	} else {
		t.Logf("dream task created: %s status=%s", dream.DreamID, dream.Status)
		dreamTask, err := client.GetMemoryDreamTask(ctx, &agentstorage.GetMemoryDreamTaskRequest{
			MemoryStoreName: store,
			DreamID:         dream.DreamID,
		})
		if err != nil {
			t.Errorf("GetMemoryDreamTask: %v", err)
		} else if dreamTask.DreamID != dream.DreamID {
			t.Errorf("GetMemoryDreamTask dreamId = %q", dreamTask.DreamID)
		}

		dreamTasks, err := client.ListMemoryDreamTasks(ctx, &agentstorage.ListMemoryDreamTasksRequest{
			MemoryStoreName: store,
			Scope:           agentstorage.Scope{AppID: conversationScope.AppID, TenantID: "*", AgentID: "*", RunID: "*"},
			Limit:           10,
		})
		if err != nil {
			t.Errorf("ListMemoryDreamTasks: %v", err)
		} else if len(dreamTasks.Tasks) == 0 {
			t.Error("ListMemoryDreamTasks returned no tasks")
		}

		if _, err := client.ListMemoryDreamActions(ctx, &agentstorage.ListMemoryDreamActionsRequest{
			MemoryStoreName: store,
			DreamID:         dream.DreamID,
			Limit:           10,
		}); err != nil {
			t.Errorf("ListMemoryDreamActions: %v", err)
		}

		cancelled, err := client.CancelMemoryDreamTask(ctx, &agentstorage.CancelMemoryDreamTaskRequest{
			MemoryStoreName: store,
			DreamID:         dream.DreamID,
		})
		if err != nil {
			// The task may finish before the cancel lands; both are acceptable.
			t.Logf("CancelMemoryDreamTask: %v", err)
		} else {
			t.Logf("dream task cancelled: status=%s taskType=%s", cancelled.Status, cancelled.TaskType)
		}
	}

	// --- Store deletion ---
	deletedStore, err := client.DeleteMemoryStore(ctx, &agentstorage.DeleteMemoryStoreRequest{MemoryStoreName: store})
	if err != nil {
		t.Fatalf("DeleteMemoryStore: %v", err)
	}
	if !deletedStore.Deleted {
		t.Errorf("DeleteMemoryStore = %+v", deletedStore)
	}

	_, err = client.GetMemoryStore(ctx, &agentstorage.GetMemoryStoreRequest{MemoryStoreName: store})
	if status, _ := serviceStatus(err); status != 404 {
		t.Errorf("GetMemoryStore after delete: err = %v, want 404", err)
	}
	t.Log("e2e lifecycle complete")
}
