from datetime import datetime, timezone

import pytest

from tablestore_agent_storage.auth import (
    Credentials,
    RequestContext,
    SignV2,
    SignV4,
    build_request_headers,
    sort_and_encode_query,
)
from tablestore_agent_storage.errors import OTSClientError


def test_sign_v2_matches_cross_language_fixture():
    context = RequestContext(Credentials("test_id", "test_key", "test_token"))
    signer = SignV2()
    headers = {"x-ots-test": "test"}

    signer.add_request_signature("test_query", headers, context)

    assert headers["x-ots-signature"] == "QDhzLv7VESBJtYQY4Li0IhSUOdg="
    assert (
        signer.make_response_signature(
            "test_query",
            {"x-ots-test": "test"},
            signer.get_signing_key(context),
        )
        == "UjJK/SWed0n9o6JYxvApHGaQABo="
    )


def test_sign_v2_respects_configured_non_utf8_encoding():
    context = RequestContext(Credentials("test_id", "秘密"))
    signer = SignV2(encoding="shift_jis")
    headers = {"x-ots-test": "日本語"}

    signer.add_request_signature("/Test", headers, context)

    assert headers["x-ots-signature"] == "A9JwKD089+mChP3AvYk9N5d/KaQ="
    assert (
        signer.make_response_signature(
            "/Test",
            {"x-ots-test": "日本語"},
            signer.get_signing_key(context),
        )
        == "BfcJ4Tj59aD3WqMTWhwz7pDe8Gk="
    )


def test_sign_v4_matches_cross_language_fixture():
    context = RequestContext(Credentials("test_id", "test_key", "test_token"))
    signer = SignV4(
        region="test-region",
        sign_date="20250410",
        auto_update_v4_sign=False,
    )
    headers = {"x-ots-test": "test"}

    signer.add_request_signature("test_query", headers, context)

    assert headers["x-ots-signregion"] == "test-region"
    assert headers["x-ots-signdate"] == "20250410"
    assert (
        headers["x-ots-signaturev4"] == "yXnOpODWaU1EYAlLP3l25ksj010uGHS7uxIt5Qiwz4o="
    )
    assert (
        signer.get_signing_key(context).decode()
        == "nToxlXrxgCm0L5J0nr/qq/GmtgN9GVBhiRLzdLaVUP0="
    )
    assert (
        signer.make_response_signature(
            "test_query",
            {"x-ots-test": "test"},
            signer.get_signing_key(context),
        )
        == "vIhaUGwv/JSg8ctLNyxbNeNv69A="
    )


def test_signed_headers_include_sts_md5_extra_headers_and_fixed_date():
    context = RequestContext(Credentials("ak", "sk", "token"))

    headers = build_request_headers(
        body=b"hello",
        query="/CreateKnowledgeBase",
        instance_name="instance",
        request_context=context,
        signer=SignV2(),
        user_agent="test-agent",
        extra_headers={"x-ots-extra": "1"},
        now=datetime(2025, 4, 10, 1, 2, 3, 456000, tzinfo=timezone.utc),
    )

    assert headers["x-ots-date"] == "2025-04-10T01:02:03.000Z"
    assert headers["x-ots-apiversion"] == "2015-12-31"
    assert headers["x-ots-accesskeyid"] == "ak"
    assert headers["x-ots-instancename"] == "instance"
    assert headers["x-ots-contentmd5"] == "XUFAKrxLKna5cZ2REBfFkg=="
    assert headers["x-ots-ststoken"] == "token"
    assert headers["x-ots-extra"] == "1"
    assert headers["User-Agent"] == "test-agent"


def test_query_parameters_are_sorted_by_key_and_value():
    assert sort_and_encode_query("z=2&a=2&a=1") == "a=1&a=2&z=2"


def test_sign_v4_requires_region():
    with pytest.raises(OTSClientError, match="region is not str or is empty"):
        SignV4(region="")
