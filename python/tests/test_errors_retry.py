from tablestore_agent_storage.errors import OTSClientError, OTSServiceError
from tablestore_agent_storage.retry import DefaultRetryPolicy, NoRetryPolicy


def test_service_error_preserves_former_shape():
    error = OTSServiceError(409, "CONFLICT", "busy", "req-1")

    assert error.http_status == 409
    assert error.code == "CONFLICT"
    assert error.message == "busy"
    assert error.request_id == "req-1"
    assert error.get_http_status() == 409
    assert error.get_error_code() == "CONFLICT"
    assert error.get_error_message() == "busy"
    assert error.get_request_id() == "req-1"
    assert str(error) == "ErrorCode: CONFLICT, ErrorMessage: busy, RequestID: req-1"


def test_client_error_preserves_former_shape():
    error = OTSClientError("bad response", 502)

    assert error.message == "bad response"
    assert error.get_http_status() == 502
    assert error.get_error_message() == "bad response"
    assert str(error) == "bad response"


def test_default_retry_policy_retries_throttling_with_bounded_delay(monkeypatch):
    monkeypatch.setattr("tablestore_agent_storage.retry.random.random", lambda: 0.0)
    policy = DefaultRetryPolicy()
    error = OTSServiceError(503, "OTSServerBusy", "busy")

    assert policy.should_retry(0, error, "CreateKnowledgeBase") is True
    assert policy.get_retry_delay(0, error, "CreateKnowledgeBase") == 0.25
    assert (
        policy.should_retry(policy.max_retry_times, error, "CreateKnowledgeBase")
        is False
    )


def test_no_retry_policy_never_retries():
    error = OTSServiceError(503, "OTSServerBusy", "busy")
    policy = NoRetryPolicy()

    assert policy.should_retry(0, error, "CreateKnowledgeBase") is False
    assert policy.get_retry_delay(0, error, "CreateKnowledgeBase") == 0
