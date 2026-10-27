# Aliyun Tablestore Agent Storage Python SDK

A Python SDK for agent storage with support for Aliyun OSS and OTS, featuring:

- knowledge base management
- document management
- chunk-level listing and update
- retrieval with metadata filters

## Installation

```bash
pip install tablestore-agent-storage
```

or install from source:

```bash
pip install -e .
```

## Quick Start

```python
from tablestore_agent_storage import (
    AgentStorageClient,
    UpdateKnowledgeBaseRequest,
    UpdateDocumentRequest,
    ListChunksRequest,
    UpdateChunkItem,
    UpdateChunksRequest,
)

# Initialize client with unified credentials
client = AgentStorageClient(
    access_key_id='your_access_key_id',
    access_key_secret='your_access_key_secret',
    oss_endpoint='https://oss-cn-hangzhou.aliyuncs.com',
    oss_bucket_name='your_bucket_name',
    ots_endpoint='https://your_instance.cn-hangzhou.ots.aliyuncs.com',
    ots_instance_name='your_instance_name'
)

# Create a knowledge base
response = client.create_knowledge_base({
    'knowledgeBaseName': 'my_knowledge_base',
    'description': 'My first knowledge base'
})

# Update knowledge base
client.update_knowledge_base(
    UpdateKnowledgeBaseRequest(
        knowledge_base_name="my_knowledge_base",
        description="Updated description",
        tags=["production", "v2"]
    )
)

# Add a document with OSS key
client.add_documents({
    'knowledgeBaseName': 'my_knowledge_base',
    'documents': [{
        'ossKey': 'oss://your-bucket/path/to/file.pdf',
        'metadata': {'author': 'aliyun'}
    }]
})

# Or add a document by uploading a local file
client.upload_documents({
    'knowledgeBaseName': 'my_knowledge_base',
    'documents': [{
        'filePath': '/path/to/local/file.pdf',
        'metadata': {'author': 'aliyun'}
    }]
})

# Update document metadata
client.update_document(
    UpdateDocumentRequest(
        knowledge_base_name="my_knowledge_base",
        doc_id="doc_123",
        metadata={"author": "new_author"}
    )
)

# List chunks
chunk_list = client.list_chunks(
    ListChunksRequest(
        knowledge_base_name="my_knowledge_base",
        doc_id="doc_123",
        max_results=10
    )
)

# Update chunks
client.update_chunks(
    UpdateChunksRequest(
        knowledge_base_name="my_knowledge_base",
        chunks=[
            UpdateChunkItem(chunk_id=1, doc_id="doc_123", title="new title"),
            UpdateChunkItem(chunk_id=2, doc_id="doc_123", content="new content"),
        ],
    )
)

# Search/Retrieve
results = client.retrieve({
    'knowledgeBaseName': 'knowledgeBaseName',
    "retrievalQuery": {
        "text": "text to search",
        "type": "TEXT"
    }
})
```

## API Reference

### Knowledge Base Operations

- `create_knowledge_base(request)` - Create new knowledge base
- `list_knowledge_base(request)` - List all knowledge bases
- `describe_knowledge_base(request)` - Get knowledge base details
- `update_knowledge_base(request)` - Update knowledge base description/tags/retrieval config
- `delete_knowledge_base(request)` - Delete knowledge base

### Document Operations

- `add_documents(request)` - Add a document to a knowledge base (requires OSS key)
- `upload_documents(request)` - Add a document by uploading a local file (automatically uploads to OSS)
- `update_document(request)` - Update document metadata by `docId` or `ossKey`
- `list_documents(request)` - List documents in knowledge base
- `get_document(request)` - Get document details
- `delete_documents(request)` - Delete documents

### Chunk Operations

- `list_chunks(request)` - List chunks by `docId`/`ossKey` with pagination support
- `update_chunks(request)` - Batch update chunk title/content/status

### Retrieval Operations

- `retrieve(request)` - Perform vector search/retrieval

### Item Operations (file-memory stores)

Memory stores in `filemem` mode (and the read-only `file+ots` projection)
expose file items through the item APIs. Every request carries
`"type": "memoryfile"`; the SDK fills it automatically when omitted
(constant `ITEM_TYPE_MEMORY_FILE`).

Requests are plain dicts. `add_item`, `get_item`, `update_item`, exact
`delete_item`, `get_item_version`, and `redact_item_version` require a fully
specified four-part `scope`. Only `list_items`, `list_item_versions`, and
ranged `delete_item` accept the left-prefix `*` wildcard (`appId`/`tenantId`
stay exact, and once a segment is `*` every later segment must be `*`). The
`view` field selects the payload shape: `"basic"` (default, no content) or
`"full"` (constants `ITEM_VIEW_BASIC` / `ITEM_VIEW_FULL`).

- `add_item(request)` - Create an item (`path` starting with `/`, required `content`, optional `sessionId`, `view`)
- `list_items(request)` - List items (`pathPrefix` starting and ending with `/`, `depth` 0/1, `view`, `limit`, `nextToken`); `depth=1` rolls deeper directories into `memoryfile_prefix` entries (`ITEM_TYPE_MEMORY_FILE_PREFIX`)
- `get_item(request)` - Get one item in a full scope with exactly one of `itemId` or `path`; optional `view="full"` adds content (default `basic` is metadata-only)
- `update_item(request)` - Update the item addressed by required `itemId`; at least one of `content` or target `path` is required and both may be combined (`precondition.expectedVersionId`, `overwrite` with `path`, `sessionId`, and `view` are optional)
- `delete_item(request)` - Delete one item in a full scope with exactly one of `itemId` or `path` (optional `precondition.expectedVersionId` and `sessionId`), or delete a wildcard scope asynchronously; ranged delete accepts optional segment-aligned `pathPrefix`/`sessionId`, forbids `itemId`/`path`/`precondition`, and returns HTTP 200 with `{type, taskId, status: "pending"}`
- `list_item_versions(request)` - List immutable versions, newest first (`itemId`, `sessionId`, `apiKeyId`, `createdAtGte`/`createdAtLte`, `operation`, `view`, `limit`, `nextToken` all optional)
- `get_item_version(request)` - Get one version by self-contained `versionId` (`view` optional); redacted versions never return content
- `redact_item_version(request)` - Irreversibly clear a version's content/sha/size/path by `versionId` (optional `sessionId`); idempotent

```python
scoped_item = {
    "memoryStoreName": "agent_memory",
    "scope": {"appId": "a", "tenantId": "t", "agentId": "ag", "runId": "r"},
}

client.add_item({
    **scoped_item,
    "path": "/notes/a.md",
    "content": "hello",
})
item = client.get_item({
    **scoped_item,
    "path": "/notes/a.md",
    "view": "full",
})
updated = client.update_item({
    **scoped_item,
    "itemId": item["itemId"],
    "content": "updated",
    "path": "/notes/b.md",  # Target path; itemId is the source address.
    "precondition": {"expectedVersionId": item["latestVersionId"]},
    "view": "full",
})
client.delete_item({
    **scoped_item,
    "itemId": updated["itemId"],
    "precondition": {"expectedVersionId": updated["latestVersionId"]},
})
```

## Update Request Semantics

`UpdateKnowledgeBaseRequest`, `UpdateDocumentRequest`, and `UpdateChunkItem` use an internal UNSET sentinel:

- field not provided: keep current value unchanged
- field explicitly set to `None`: clear this field on server side

This makes partial update behavior explicit and predictable.

## Configuration

The SDK supports API Key authentication or signed AK/STS requests. All Agent
Storage requests are implemented inside this package and do not depend on the
general Tablestore Python SDK exposing matching methods.

Core configuration parameters:

- `access_key_id`: Your Aliyun access key ID (shared by OSS and OTS)
- `access_key_secret`: Your Aliyun access key secret (shared by OSS and OTS)
- `sts_token`: Optional STS security token
- `api_key`: API Key for OTS requests; requires an HTTPS endpoint
- `oss_endpoint`: OSS service endpoint
- `oss_bucket_name`: OSS bucket name
- `ots_endpoint`: OTS service endpoint
- `ots_instance_name`: OTS instance name

Additional keyword options remain available:

- `credentials_provider`: Refreshing provider with a `get_credentials()` method
- `region`: Enables SignV4 for the specified region; SignV2 is the default
- `sign_version`: Explicitly select `v2` or `v4`
- `socket_timeout`: Per-request timeout in seconds
- `max_connection`: HTTP connection-pool size
- `ssl_version`: Minimum TLS version
- `extra_headers`: Headers attached to every OTS request
- `retry_policy`: Object implementing `should_retry()` and `get_retry_delay()`

## Examples

See the `examples/` directory for more detailed usage examples.

## Dependencies

- Python >= 3.8
- oss2 >= 2.18.0
- urllib3 >= 1.26, < 3

# Contact

- [阿里云 Tablestore 官方网站](http://www.aliyun.com/product/ots)
- [阿里云官网联系方式](https://help.aliyun.com/document_detail/61890.html)
- [阿里云 Tablestore 官方文档](https://help.aliyun.com/zh/tablestore/product-overview)
