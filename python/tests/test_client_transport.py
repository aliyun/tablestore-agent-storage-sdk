import sys
import types

import pytest

fake_oss2 = types.ModuleType("oss2")
fake_oss2.Auth = object
fake_oss2.StsAuth = object
fake_oss2.Bucket = object
sys.modules.setdefault("oss2", fake_oss2)

from tablestore_agent_storage.auth import SignV4  # noqa: E402
from tablestore_agent_storage.client import (  # noqa: E402
    ITEM_TYPE_MEMORY_FILE,
    ITEM_TYPE_MEMORY_FILE_PREFIX,
    ITEM_VIEW_BASIC,
    ITEM_VIEW_FULL,
    AgentStorageClient,
)
from tablestore_agent_storage.ots_json_client import OTSJSONClient  # noqa: E402


class EmptyTransport:
    def request(self, method, url, **kwargs):
        raise AssertionError("network should not be used during construction")


class RecordingOTSClient:
    def __init__(self):
        self.calls = []

    def request(self, action, payload):
        self.calls.append((action, payload))
        return {"action": action}

def facade_with_recorder():
    client = AgentStorageClient.__new__(AgentStorageClient)
    client.ots_client = RecordingOTSClient()
    client.oss_client = None
    client.oss_bucket_name = None
    client.ots_instance_name = "instance"
    return client


def test_ak_constructor_uses_internal_ots_json_client_and_forwards_options():
    transport = EmptyTransport()
    retry_policy = object()
    client = AgentStorageClient(
        access_key_id="ak",
        access_key_secret="sk",
        sts_token="token",
        ots_endpoint="https://example.com",
        ots_instance_name="instance",
        region="cn-hangzhou",
        sign_version="v4",
        socket_timeout=12,
        max_connection=7,
        extra_headers={"x-ots-extra": "1"},
        retry_policy=retry_policy,
        _ots_transport=transport,
    )

    assert isinstance(client.ots_client, OTSJSONClient)
    assert isinstance(client.ots_client.signer, SignV4)
    assert client.ots_client.socket_timeout == 12
    assert client.ots_client.max_connection == 7
    assert client.ots_client.extra_headers == {"x-ots-extra": "1"}
    assert client.ots_client.retry_policy is retry_policy
    assert client.ots_client.transport is transport


def test_api_key_constructor_uses_same_internal_client():
    client = AgentStorageClient(
        api_key="key",
        ots_endpoint="https://example.com",
        ots_instance_name="instance",
        _ots_transport=EmptyTransport(),
    )

    assert isinstance(client.ots_client, OTSJSONClient)
    assert client.ots_client.api_key == "key"


class StaticProvider:
    def get_credentials(self):
        return types.SimpleNamespace(
            access_key_id="ak",
            access_key_secret="sk",
            security_token="token",
        )


def test_credentials_provider_can_authenticate_without_static_keys():
    provider = StaticProvider()
    client = AgentStorageClient(
        ots_endpoint="https://example.com",
        ots_instance_name="instance",
        credentials_provider=provider,
        _ots_transport=EmptyTransport(),
    )

    assert client.ots_client.credentials_provider is provider


@pytest.mark.parametrize(
    "method,action",
    [
        ("create_knowledge_base", "CreateKnowledgeBase"),
        ("delete_knowledge_base", "DeleteKnowledgeBase"),
        ("describe_knowledge_base", "DescribeKnowledgeBase"),
        ("list_knowledge_base", "ListKnowledgeBase"),
        ("update_knowledge_base", "UpdateKnowledgeBase"),
        ("add_documents", "AddDocuments"),
        ("update_document", "UpdateDocument"),
        ("get_document", "GetDocument"),
        ("list_documents", "ListDocuments"),
        ("retrieve", "Retrieve"),
        ("list_chunks", "ListChunks"),
        ("update_chunks", "UpdateChunks"),
        ("create_memory_store", "CreateMemoryStore"),
        ("get_memory_store", "GetMemoryStore"),
        ("list_memory_stores", "ListMemoryStores"),
        ("update_memory_store", "UpdateMemoryStore"),
        ("delete_memory_store", "DeleteMemoryStore"),
        ("add_memories", "AddMemories"),
        ("search_memories", "SearchMemories"),
        ("list_memories", "ListMemories"),
        ("get_memory", "GetMemory"),
        ("update_memory", "UpdateMemory"),
        ("delete_memory", "DeleteMemory"),
        ("list_memory_store_messages", "ListMemoryStoreMessages"),
        ("list_memory_store_requests", "ListMemoryStoreRequests"),
        ("get_memory_task", "GetMemoryTask"),
        ("list_memory_tasks", "ListMemoryTasks"),
        ("list_memory_store_scopes", "ListMemoryStoreScopes"),
        ("create_memory_dream_task", "CreateMemoryDreamTask"),
        ("get_memory_dream_task", "GetMemoryDreamTask"),
        ("list_memory_dream_tasks", "ListMemoryDreamTasks"),
        ("cancel_memory_dream_task", "CancelMemoryDreamTask"),
        ("list_memory_dream_actions", "ListMemoryDreamActions"),
        ("apply_memory_dream_actions", "ApplyMemoryDreamActions"),
    ],
)
def test_public_methods_use_explicit_actions(method, action):
    client = facade_with_recorder()

    result = getattr(client, method)({})

    assert result["action"] == action
    assert client.ots_client.calls == [(action, {})]


ITEM_METHOD_ACTIONS = [
    ("add_item", "AddItem"),
    ("list_items", "ListItems"),
    ("get_item", "GetItem"),
    ("update_item", "UpdateItem"),
    ("delete_item", "DeleteItem"),
    ("list_item_versions", "ListItemVersions"),
    ("get_item_version", "GetItemVersion"),
    ("redact_item_version", "RedactItemVersion"),
]


@pytest.mark.parametrize("method,action", ITEM_METHOD_ACTIONS)
def test_item_methods_use_explicit_actions_and_fill_default_type(method, action):
    client = facade_with_recorder()
    request = {"memoryStoreName": "store"}

    result = getattr(client, method)(request)

    assert result["action"] == action
    called_action, payload = client.ots_client.calls[0]
    assert called_action == action
    assert payload["type"] == "memoryfile"
    assert payload["memoryStoreName"] == "store"
    # The caller's dict is copied, never mutated.
    assert request == {"memoryStoreName": "store"}


@pytest.mark.parametrize("method,action", ITEM_METHOD_ACTIONS)
def test_item_methods_pass_explicit_type_through(method, action):
    client = facade_with_recorder()

    getattr(client, method)({"type": "othertype"})

    _, payload = client.ots_client.calls[0]
    assert payload["type"] == "othertype"


@pytest.mark.parametrize("method,action", ITEM_METHOD_ACTIONS)
def test_item_methods_fill_type_when_empty(method, action):
    client = facade_with_recorder()

    getattr(client, method)({"type": ""})

    _, payload = client.ots_client.calls[0]
    assert payload["type"] == "memoryfile"


@pytest.mark.parametrize("method,action", ITEM_METHOD_ACTIONS)
def test_item_methods_tolerate_none_request(method, action):
    """None requests degrade to {"type": "memoryfile"}, matching Go/TS."""
    client = facade_with_recorder()

    getattr(client, method)(None)

    called_action, payload = client.ots_client.calls[0]
    assert called_action == action
    assert payload == {"type": "memoryfile"}


def test_item_type_constant_value():
    assert ITEM_TYPE_MEMORY_FILE == "memoryfile"


def test_item_prefix_type_constant_value():
    assert ITEM_TYPE_MEMORY_FILE_PREFIX == "memoryfile_prefix"


def test_item_view_constant_values():
    assert ITEM_VIEW_BASIC == "basic"
    assert ITEM_VIEW_FULL == "full"


def test_update_item_content_branch_forwards_fields():
    client = facade_with_recorder()

    client.update_item(
        {
            "memoryStoreName": "store",
            "itemId": "m1",
            "content": "",
            "precondition": {"expectedVersionId": "mv1"},
            "sessionId": "session-1",
        }
    )

    action, payload = client.ots_client.calls[0]
    assert action == "UpdateItem"
    assert payload["itemId"] == "m1"
    assert payload["content"] == ""
    assert payload["precondition"] == {"expectedVersionId": "mv1"}
    assert payload["sessionId"] == "session-1"
    assert "path" not in payload


def test_update_item_rename_branch_forwards_fields():
    client = facade_with_recorder()

    client.update_item(
        {
            "memoryStoreName": "store",
            "itemId": "m1",
            "path": "/notes/b.md",
            "overwrite": True,
            "precondition": {"expectedVersionId": "mv2"},
        }
    )

    action, payload = client.ots_client.calls[0]
    assert action == "UpdateItem"
    assert payload["itemId"] == "m1"
    assert payload["path"] == "/notes/b.md"
    assert payload["overwrite"] is True
    assert payload["precondition"] == {"expectedVersionId": "mv2"}
    assert "content" not in payload


def test_get_item_forwards_item_id_and_view():
    client = facade_with_recorder()

    client.get_item({"itemId": "m1", "view": "full"})

    action, payload = client.ots_client.calls[0]
    assert action == "GetItem"
    assert payload["itemId"] == "m1"
    assert payload["view"] == "full"
    assert payload["type"] == "memoryfile"


def test_get_item_forwards_path_only():
    client = facade_with_recorder()

    client.get_item({"path": "/notes/a.md"})

    action, payload = client.ots_client.calls[0]
    assert action == "GetItem"
    assert payload["path"] == "/notes/a.md"
    assert "itemId" not in payload


def test_update_item_composite_branch_forwards_fields():
    client = facade_with_recorder()

    client.update_item(
        {
            "memoryStoreName": "store",
            "itemId": "m1",
            "content": "new",
            "path": "/notes/b.md",
        }
    )

    action, payload = client.ots_client.calls[0]
    assert action == "UpdateItem"
    assert payload["itemId"] == "m1"
    assert payload["content"] == "new"
    assert payload["path"] == "/notes/b.md"


def test_list_items_forwards_depth_and_view():
    client = facade_with_recorder()

    client.list_items({"pathPrefix": "/notes/", "depth": 1, "view": "full", "limit": 20})

    action, payload = client.ots_client.calls[0]
    assert action == "ListItems"
    assert payload["pathPrefix"] == "/notes/"
    assert payload["depth"] == 1
    assert payload["view"] == "full"
    assert payload["limit"] == 20


def test_delete_item_ranged_forwards_wildcard_scope():
    client = facade_with_recorder()

    client.delete_item(
        {
            "scope": {"appId": "a", "tenantId": "t", "agentId": "*", "runId": "*"},
            "pathPrefix": "/notes/",
            "sessionId": "s1",
        }
    )

    action, payload = client.ots_client.calls[0]
    assert action == "DeleteItem"
    assert payload["scope"]["agentId"] == "*"
    assert payload["pathPrefix"] == "/notes/"
    assert payload["sessionId"] == "s1"
    assert "precondition" not in payload


def test_delete_item_exact_forwards_resource_version_precondition():
    client = facade_with_recorder()

    client.delete_item(
        {
            "memoryStoreName": "store",
            "itemId": "m1",
            "precondition": {"expectedVersionId": "mv3"},
            "sessionId": "session-1",
        }
    )

    action, payload = client.ots_client.calls[0]
    assert action == "DeleteItem"
    assert payload["itemId"] == "m1"
    assert payload["precondition"] == {"expectedVersionId": "mv3"}
    assert payload["sessionId"] == "session-1"


def test_list_item_versions_forwards_filters():
    client = facade_with_recorder()

    client.list_item_versions(
        {
            "itemId": "m1",
            "sessionId": "s1",
            "createdAtGte": "2026-07-01T00:00:00Z",
            "createdAtLte": "2026-07-31T23:59:59Z",
            "operation": "modified",
            "view": "full",
        }
    )

    action, payload = client.ots_client.calls[0]
    assert action == "ListItemVersions"
    assert payload["sessionId"] == "s1"
    assert payload["createdAtGte"] == "2026-07-01T00:00:00Z"
    assert payload["createdAtLte"] == "2026-07-31T23:59:59Z"
    assert payload["view"] == "full"


def test_get_item_version_forwards_version_id_only():
    client = facade_with_recorder()

    client.get_item_version({"versionId": "mv1", "view": "full"})

    action, payload = client.ots_client.calls[0]
    assert action == "GetItemVersion"
    assert payload["versionId"] == "mv1"
    assert payload["view"] == "full"


def test_redact_item_version_forwards_version_id_only():
    client = facade_with_recorder()

    client.redact_item_version({"versionId": "mv1", "sessionId": "redactor-1"})

    action, payload = client.ots_client.calls[0]
    assert action == "RedactItemVersion"
    assert payload["versionId"] == "mv1"
    assert payload["sessionId"] == "redactor-1"


def test_delete_documents_uses_explicit_action():
    client = facade_with_recorder()
    request = {
        "knowledgeBaseName": "kb",
        "documents": [{"docId": "doc"}],
    }

    result = client.delete_documents(request)

    assert result["action"] == "DeleteDocuments"
    assert client.ots_client.calls == [("DeleteDocuments", request)]


class RecordingOSSClient:
    def __init__(self):
        self.uploads = []

    def put_object(self, key, stream):
        self.uploads.append((key, stream.read()))

    def batch_delete_objects(self, keys):
        raise AssertionError("cleanup should not run on success")


def test_upload_documents_finishes_with_explicit_add_documents(tmp_path):
    source = tmp_path / "document.txt"
    source.write_bytes(b"content")
    client = facade_with_recorder()
    client.oss_client = RecordingOSSClient()
    client.oss_bucket_name = "bucket"
    request = {
        "knowledgeBaseName": "kb",
        "documents": [{"filePath": str(source)}],
    }

    result = client.upload_documents(request)

    assert result["action"] == "AddDocuments"
    action, payload = client.ots_client.calls[0]
    assert action == "AddDocuments"
    assert payload["documents"][0]["ossKey"].startswith("oss://bucket/")
    assert client.oss_client.uploads[0][1] == b"content"
