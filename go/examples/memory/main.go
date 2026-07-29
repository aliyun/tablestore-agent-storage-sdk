// Example: memory store lifecycle — create a store, write memories, search
// them, inspect the async task and clean up.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

func main() {
	client, err := agentstorage.NewClient(agentstorage.Config{
		Endpoint:        os.Getenv("AGENT_STORAGE_ENDPOINT"),
		InstanceName:    os.Getenv("AGENT_STORAGE_INSTANCE"),
		AccessKeyID:     os.Getenv("AGENT_STORAGE_AK"),
		AccessKeySecret: os.Getenv("AGENT_STORAGE_SK"),
		APIKey:          os.Getenv("AGENT_STORAGE_API_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	scope := &agentstorage.Scope{AppID: "app-001", TenantID: "user-001"}

	if _, err := client.CreateMemoryStore(ctx, &agentstorage.CreateMemoryStoreRequest{
		MemoryStoreName: "agent_memory_example",
		Description:     "Go SDK example store",
	}); err != nil {
		log.Printf("create store: %v", err)
	}

	added, err := client.AddMemories(ctx, &agentstorage.AddMemoriesRequest{
		MemoryStoreName: "agent_memory_example",
		Scope:           scope,
		Messages: []agentstorage.MessagePayload{
			{Role: "user", Content: "我喜欢喝咖啡"},
			{Role: "assistant", Content: "好的，我记住了"},
		},
		Sync: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("added: status=%s accepted=%d\n", added.Status, added.AcceptedMessages)

	hits, err := client.SearchMemories(ctx, &agentstorage.SearchMemoriesRequest{
		MemoryStoreName: "agent_memory_example",
		Scope:           &agentstorage.Scope{AppID: "app-001", TenantID: "user-001", AgentID: "*", RunID: "*"},
		Query:           "用户喜欢什么饮品",
		TopK:            5,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, hit := range hits.Results {
		fmt.Printf("hit: %.3f %s\n", hit.Score, hit.Unit.Text)
	}
}
