import base64
import hashlib
from datetime import datetime, timezone

import pytest

from tablestore_agent_storage.auth import SignV2
from tablestore_agent_storage.errors import OTSClientError, OTSServiceError
from tablestore_agent_storage.ots_json_client import OTSJSONClient

FIXED_NOW = datetime(2025, 4, 10, 1, 2, 3, tzinfo=timezone.utc)


def fixed_clock():
    return FIXED_NOW


class FakeResponse:
    def __init__(self, status=200, data=b"{}", headers=None):
        self.status = status
        self.data = data
        self.headers = headers or {}


class FakeTransport:
    def __init__(self, responses):
        self.responses = list(responses)
        self.calls = []

    def request(self, method, url, **kwargs):
        self.calls.append((method, url, kwargs))
        response = self.responses.pop(0)
        if isinstance(response, Exception):
            raise response
        return response


def signed_response_headers(
    payload,
    query="/CreateKnowledgeBase",
    access_key_id="ak",
    access_key_secret="sk",
    response_date="2025-04-10T01:02:03.000Z",
):
    headers = {
        "x-ots-contentmd5": base64.b64encode(hashlib.md5(payload).digest()).decode(),
        "x-ots-requestid": "req-1",
        "x-ots-date": response_date,
        "x-ots-contenttype": "application/json",
    }
    signature = SignV2().make_response_signature(
        query,
        headers,
        access_key_secret.encode(),
    )
    headers["authorization"] = "OTS %s:%s" % (access_key_id, signature)
    return headers


def make_signed_client(response, **kwargs):
    transport = FakeTransport([response])
    client = OTSJSONClient(
        "https://example.com/",
        "instance",
        access_key_id="ak",
        access_key_secret="sk",
        sts_token="token",
        transport=transport,
        clock=fixed_clock,
        **kwargs,
    )
    return client, transport


def test_api_key_request_uses_unified_transport():
    transport = FakeTransport([FakeResponse(data=b'{"ok":true}')])
    client = OTSJSONClient(
        "https://example.com/",
        "instance",
        api_key="key",
        transport=transport,
    )

    assert client.request("ListItems", {"memoryStoreName": "store"}) == {
        "ok": True
    }

    method, url, kwargs = transport.calls[0]
    assert (method, url) == ("POST", "https://example.com/ListItems")
    assert kwargs["headers"]["x-ots-apikey"] == "key"
    assert kwargs["headers"]["x-ots-instancename"] == "instance"
    assert kwargs["headers"]["Content-Type"] == "application/json"
    assert kwargs["headers"]["User-Agent"] == "tablestore-agent-storage-python/1.0.10"
    assert kwargs["body"] == b'{"memoryStoreName":"store"}'
    assert kwargs["retries"] is False


def test_api_key_rejects_http_endpoint():
    with pytest.raises(OTSClientError, match="requires HTTPS"):
        OTSJSONClient("http://example.com", "instance", api_key="key")


def test_default_transport_blocks_when_connection_pool_is_full(monkeypatch):
    pool_options = {}
    pool = object()

    def create_pool(**options):
        pool_options.update(options)
        return pool

    monkeypatch.setattr(
        "tablestore_agent_storage.ots_json_client.urllib3.PoolManager",
        create_pool,
    )

    client = OTSJSONClient("https://example.com", "instance", api_key="key")

    assert client.transport is pool
    assert pool_options["block"] is True


def test_none_max_connection_uses_default_pool_size(monkeypatch):
    pool_options = {}

    def create_pool(**options):
        pool_options.update(options)
        return object()

    monkeypatch.setattr(
        "tablestore_agent_storage.ots_json_client.urllib3.PoolManager",
        create_pool,
    )

    client = OTSJSONClient(
        "https://example.com",
        "instance",
        api_key="key",
        max_connection=None,
    )

    assert client.max_connection == 500
    assert pool_options["maxsize"] == 500


def test_client_requires_authentication():
    with pytest.raises(OTSClientError, match="Either credentials"):
        OTSJSONClient("https://example.com", "instance")


@pytest.mark.parametrize(
    "credentials,error_message",
    [
        ({"access_key_id": 123, "access_key_secret": "sk"}, "access_key_id"),
        ({"access_key_id": "ak", "access_key_secret": object()}, "access_key_secret"),
        ({"access_key_id": "", "access_key_secret": "sk"}, "access_key_id"),
        ({"access_key_id": "ak", "access_key_secret": ""}, "access_key_secret"),
    ],
)
def test_static_credentials_are_validated_during_construction(
    credentials,
    error_message,
):
    with pytest.raises(OTSClientError, match=error_message):
        OTSJSONClient(
            "https://example.com",
            "instance",
            transport=FakeTransport([]),
            **credentials,
        )


def test_signed_client_passes_configured_encoding_to_signer():
    client = OTSJSONClient(
        "https://example.com",
        "instance",
        access_key_id="ak",
        access_key_secret="sk",
        encoding="shift_jis",
        transport=FakeTransport([]),
    )

    assert client.signer.encoding == "shift_jis"


def test_signed_request_validates_response_and_sends_sts_headers():
    payload = b'{"code":"SUCCESS"}'
    client, transport = make_signed_client(
        FakeResponse(data=payload, headers=signed_response_headers(payload)),
        extra_headers={"x-ots-extra": "1"},
    )

    assert client.request("CreateKnowledgeBase", {"knowledgeBaseName": "kb"}) == {
        "code": "SUCCESS"
    }

    sent = transport.calls[0][2]["headers"]
    assert sent["x-ots-accesskeyid"] == "ak"
    assert sent["x-ots-ststoken"] == "token"
    assert sent["x-ots-extra"] == "1"
    assert "x-ots-signature" in sent


@pytest.mark.parametrize(
    "headers,error_message",
    [
        ({}, "missing in response header"),
        (
            {
                "x-ots-contentmd5": "bogus",
                "x-ots-requestid": "req-1",
                "x-ots-date": "2025-04-10T01:02:03.000Z",
                "x-ots-contenttype": "application/json",
            },
            "MD5 mismatch",
        ),
    ],
)
def test_signed_response_rejects_missing_headers_and_bad_md5(headers, error_message):
    client, _ = make_signed_client(FakeResponse(data=b"{}", headers=headers))

    with pytest.raises(OTSClientError, match=error_message):
        client.request("CreateKnowledgeBase", {})


def test_signed_response_rejects_excessive_clock_skew():
    payload = b"{}"
    headers = signed_response_headers(payload, response_date="2025-04-09T01:02:03.000Z")
    client, _ = make_signed_client(FakeResponse(data=payload, headers=headers))

    with pytest.raises(OTSClientError, match="more than 15 minutes"):
        client.request("CreateKnowledgeBase", {})


def test_signed_response_rejects_wrong_access_key():
    payload = b"{}"
    headers = signed_response_headers(payload, access_key_id="other")
    client, _ = make_signed_client(FakeResponse(data=payload, headers=headers))

    with pytest.raises(OTSClientError, match="Invalid access key id"):
        client.request("CreateKnowledgeBase", {})


def test_signed_response_rejects_wrong_signature():
    payload = b"{}"
    headers = signed_response_headers(payload)
    headers["authorization"] = "OTS ak:wrong"
    client, _ = make_signed_client(FakeResponse(data=payload, headers=headers))

    with pytest.raises(OTSClientError, match="Invalid signature"):
        client.request("CreateKnowledgeBase", {})


def test_transport_error_maps_to_client_error():
    transport = FakeTransport([OSError("network down")])
    client = OTSJSONClient(
        "https://example.com",
        "instance",
        api_key="key",
        transport=transport,
    )

    with pytest.raises(OTSClientError, match="network down"):
        client.request("CreateKnowledgeBase", {})


def test_non_2xx_maps_to_service_error():
    transport = FakeTransport(
        [
            FakeResponse(
                409,
                b'{"code":"CONFLICT","message":"busy","requestId":"body-req"}',
                {"x-ots-requestid": "header-req"},
            )
        ]
    )
    client = OTSJSONClient(
        "https://example.com",
        "instance",
        api_key="key",
        transport=transport,
    )

    with pytest.raises(OTSServiceError) as captured:
        client.request("CreateKnowledgeBase", {})

    assert captured.value.http_status == 409
    assert captured.value.code == "CONFLICT"
    assert captured.value.message == "busy"
    assert captured.value.request_id == "header-req"


def test_invalid_success_json_maps_to_client_error():
    transport = FakeTransport([FakeResponse(data=b"not-json")])
    client = OTSJSONClient(
        "https://example.com",
        "instance",
        api_key="key",
        transport=transport,
    )

    with pytest.raises(OTSClientError, match="Failed to parse JSON response"):
        client.request("CreateKnowledgeBase", {})


class FormerCredentials:
    def __init__(self, key_id, secret, token):
        self._values = key_id, secret, token

    def get_access_key_id(self):
        return self._values[0]

    def get_access_key_secret(self):
        return self._values[1]

    def get_security_token(self):
        return self._values[2]


class RefreshingProvider:
    def __init__(self):
        self.calls = 0

    def get_credentials(self):
        self.calls += 1
        return FormerCredentials("ak", "sk", "token-%d" % self.calls)


def test_credentials_provider_refreshes_before_each_request():
    payload = b'{"ok":true}'
    responses = [
        FakeResponse(
            data=payload,
            headers=signed_response_headers(payload, query="/GetMemoryStore"),
        ),
        FakeResponse(
            data=payload,
            headers=signed_response_headers(payload, query="/GetMemoryStore"),
        ),
    ]
    transport = FakeTransport(responses)
    provider = RefreshingProvider()
    client = OTSJSONClient(
        "https://example.com",
        "instance",
        credentials_provider=provider,
        transport=transport,
        clock=fixed_clock,
    )

    client.request("GetMemoryStore", {})
    client.request("GetMemoryStore", {})

    assert provider.calls == 2
    assert transport.calls[0][2]["headers"]["x-ots-ststoken"] == "token-1"
    assert transport.calls[1][2]["headers"]["x-ots-ststoken"] == "token-2"


def test_api_key_takes_precedence_over_supplied_credentials_provider():
    transport = FakeTransport([FakeResponse(data=b'{"ok":true}')])
    provider = RefreshingProvider()
    client = OTSJSONClient(
        "https://example.com",
        "instance",
        api_key="key",
        credentials_provider=provider,
        transport=transport,
    )

    client.request("GetMemoryStore", {})

    assert provider.calls == 0
    assert transport.calls[0][2]["headers"]["x-ots-apikey"] == "key"


class RetryOncePolicy:
    def __init__(self):
        self.calls = []

    def should_retry(self, retry_times, exception, api_name):
        self.calls.append((retry_times, exception.code, api_name))
        return retry_times == 0

    def get_retry_delay(self, retry_times, exception, api_name):
        assert (retry_times, exception.code, api_name) == (
            0,
            "OTSServerBusy",
            "GetMemoryStore",
        )
        return 0.25


def test_custom_retry_policy_retries_service_error_once():
    policy = RetryOncePolicy()
    sleeps = []
    transport = FakeTransport(
        [
            FakeResponse(503, b'{"code":"OTSServerBusy","message":"busy"}'),
            FakeResponse(200, b'{"ok":true}'),
        ]
    )
    client = OTSJSONClient(
        "https://example.com",
        "instance",
        api_key="key",
        retry_policy=policy,
        sleep=sleeps.append,
        transport=transport,
    )

    assert client.request("GetMemoryStore", {}) == {"ok": True}
    assert policy.calls == [(0, "OTSServerBusy", "GetMemoryStore")]
    assert sleeps == [0.25]
    assert len(transport.calls) == 2


def test_binary_file_transport_is_not_exposed():
    assert not hasattr(OTSJSONClient, "request_binary")
    assert not hasattr(OTSJSONClient, "get_memory_file")
