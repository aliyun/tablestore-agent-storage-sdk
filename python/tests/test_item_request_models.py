import os
import subprocess
import sys
from pathlib import Path

from tablestore_agent_storage.models import (
    DeleteItemRequest,
    GetItemRequest,
    ItemPrecondition,
    ItemScope,
    ItemScopedRequest,
    UpdateItemRequest,
)


PYTHON_ROOT = Path(__file__).resolve().parents[1]


def test_item_request_models_are_public_package_exports():
    code = """
import sys
import types
sys.modules['oss2'] = types.ModuleType('oss2')
import tablestore_agent_storage as sdk
names = [
    'ItemScope',
    'ItemPrecondition',
    'ItemScopedRequest',
    'GetItemRequest',
    'UpdateItemRequest',
    'DeleteItemRequest',
]
print(','.join(name for name in names if name in sdk.__all__ and hasattr(sdk, name)))
"""
    environment = dict(os.environ)
    environment["PYTHONPATH"] = str(PYTHON_ROOT / "src")

    result = subprocess.run(
        [sys.executable, "-c", code],
        check=True,
        capture_output=True,
        text=True,
        env=environment,
    )

    assert result.stdout.strip().split(",") == [
        "ItemScope",
        "ItemPrecondition",
        "ItemScopedRequest",
        "GetItemRequest",
        "UpdateItemRequest",
        "DeleteItemRequest",
    ]


def test_item_request_models_are_public_typed_dicts():
    assert ItemScope.__required_keys__ == {
        "appId",
        "tenantId",
        "agentId",
        "runId",
    }
    assert ItemPrecondition.__required_keys__ == {"expectedVersionId"}
    assert ItemScopedRequest.__required_keys__ == {"memoryStoreName", "scope"}
    assert ItemScopedRequest.__optional_keys__ == {"type"}
    assert GetItemRequest.__required_keys__ == {"memoryStoreName", "scope"}
    assert GetItemRequest.__optional_keys__ == {"type", "itemId", "path", "view"}
    assert UpdateItemRequest.__required_keys__ == {
        "memoryStoreName",
        "scope",
        "itemId",
    }
    assert UpdateItemRequest.__optional_keys__ == {
        "type",
        "content",
        "path",
        "precondition",
        "overwrite",
        "sessionId",
        "view",
    }
    assert DeleteItemRequest.__required_keys__ == {"memoryStoreName", "scope"}
    assert DeleteItemRequest.__optional_keys__ == {
        "type",
        "itemId",
        "path",
        "pathPrefix",
        "precondition",
        "sessionId",
    }
