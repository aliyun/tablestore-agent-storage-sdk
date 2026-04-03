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

## Update Request Semantics

`UpdateKnowledgeBaseRequest`, `UpdateDocumentRequest`, and `UpdateChunkItem` use an internal UNSET sentinel:

- field not provided: keep current value unchanged
- field explicitly set to `None`: clear this field on server side

This makes partial update behavior explicit and predictable.

## Configuration

The SDK requires the following configuration parameters:

- `access_key_id`: Your Aliyun access key ID (shared by OSS and OTS)
- `access_key_secret`: Your Aliyun access key secret (shared by OSS and OTS)
- `oss_endpoint`: OSS service endpoint
- `oss_bucket_name`: OSS bucket name
- `ots_endpoint`: OTS service endpoint (optional)
- `ots_instance_name`: OTS instance name (optional)

## Examples

See the `examples/` directory for more detailed usage examples.

## Dependencies

- Python >= 3.8
- oss2 >= 2.18.0
- tablestore >= 6.4.3

# Contact

- [阿里云 Tablestore 官方网站](http://www.aliyun.com/product/ots)
- [阿里云官网联系方式](https://help.aliyun.com/document_detail/61890.html)
- [阿里云 Tablestore 官方文档](https://help.aliyun.com/zh/tablestore/product-overview)
