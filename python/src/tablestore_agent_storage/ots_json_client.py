"""Unified OTS JSON transport for API Key, AK, and STS authentication."""

import json
import logging
import ssl
import time
from datetime import datetime, timezone
from typing import Any, Callable, Dict, Mapping, Optional, Tuple
from urllib.parse import urlsplit

import urllib3

from tablestore_agent_storage.auth import (
    Credentials,
    RequestContext,
    SignV2,
    SignV4,
    StaticCredentialsProvider,
    build_request_headers,
    md5_base64,
)
from tablestore_agent_storage.errors import OTSClientError, OTSServiceError
from tablestore_agent_storage.retry import DefaultRetryPolicy

DEFAULT_USER_AGENT = "tablestore-agent-storage-python/1.0.10"
SIGNED_DEFAULT_TIMEOUT = 50
API_KEY_DEFAULT_TIMEOUT = 30
DEFAULT_MAX_CONNECTION = 500
MAX_RESPONSE_CLOCK_SKEW_SECONDS = 15 * 60


class OTSJSONClient:
    """Send Agent Storage JSON actions without the general Tablestore SDK."""

    def __init__(
        self,
        endpoint: str,
        instance_name: str,
        access_key_id: Optional[str] = None,
        access_key_secret: Optional[str] = None,
        sts_token: Optional[str] = None,
        api_key: Optional[str] = None,
        credentials_provider=None,
        region: Optional[str] = None,
        sign_version: Optional[str] = None,
        socket_timeout: Optional[float] = None,
        max_connection: int = DEFAULT_MAX_CONNECTION,
        ssl_version=None,
        extra_headers: Optional[Mapping[str, str]] = None,
        logger_name: Optional[str] = None,
        encoding: str = "utf-8",
        retry_policy=None,
        sign_date: Optional[str] = None,
        auto_update_v4_sign: Optional[bool] = None,
        transport=None,
        clock: Optional[Callable[[], datetime]] = None,
        sleep: Optional[Callable[[float], None]] = None,
        **kwargs,
    ):
        del kwargs
        self.endpoint = self._validate_endpoint(endpoint)
        if not isinstance(instance_name, str) or not instance_name:
            raise OTSClientError("instance_name is not str or is empty.")
        self.instance_name = instance_name
        self.api_key = api_key
        if api_key and not self.endpoint.startswith("https://"):
            raise OTSClientError(
                "API key authentication requires HTTPS. Endpoint must start with https://."
            )

        if api_key:
            self.credentials_provider = None
        elif credentials_provider is not None:
            self.credentials_provider = credentials_provider
        elif access_key_id is not None or access_key_secret is not None:
            if not isinstance(access_key_id, str) or not access_key_id:
                raise OTSClientError("access_key_id is not str or is empty.")
            if not isinstance(access_key_secret, str) or not access_key_secret:
                raise OTSClientError("access_key_secret is not str or is empty.")
            self.credentials_provider = StaticCredentialsProvider(
                Credentials(access_key_id, access_key_secret, sts_token)
            )
        else:
            raise OTSClientError(
                "Either credentials (accessKeyId/accessKeySecret) or apiKey must be provided."
            )

        normalized_sign_version = (sign_version or ("v4" if region else "v2")).lower()
        if normalized_sign_version == "v4":
            self.signer = SignV4(
                region=region or "",
                sign_date=sign_date,
                auto_update_v4_sign=auto_update_v4_sign,
                clock=clock,
                encoding=encoding,
            )
        elif normalized_sign_version == "v2":
            self.signer = SignV2(encoding)
        else:
            raise OTSClientError(
                "unsupported sign version: %s" % normalized_sign_version
            )

        self.socket_timeout = socket_timeout
        if self.socket_timeout is None:
            self.socket_timeout = (
                API_KEY_DEFAULT_TIMEOUT if api_key else SIGNED_DEFAULT_TIMEOUT
            )
        self.max_connection = (
            DEFAULT_MAX_CONNECTION if max_connection is None else max_connection
        )
        self.extra_headers = dict(extra_headers or {})
        self.encoding = encoding
        self.retry_policy = retry_policy or DefaultRetryPolicy()
        self.logger = logging.getLogger(logger_name or __name__)
        self.clock = clock or (lambda: datetime.now(timezone.utc))
        self.sleep = sleep or time.sleep
        self.user_agent = DEFAULT_USER_AGENT
        self.transport = transport or self._create_transport(
            self.max_connection, ssl_version
        )

    @staticmethod
    def _validate_endpoint(endpoint: str) -> str:
        if not isinstance(endpoint, str) or not endpoint:
            raise OTSClientError("endpoint is not str or is empty.")
        parsed = urlsplit(endpoint)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc:
            raise OTSClientError(
                "protocol of endpoint must be 'http' or 'https', for example "
                "https://instance.cn-hangzhou.ots.aliyuncs.com."
            )
        return endpoint.rstrip("/")

    @staticmethod
    def _create_transport(max_connection: int, ssl_version):
        options = {
            "maxsize": max_connection,
            "cert_reqs": "CERT_REQUIRED",
            "block": True,
        }
        if ssl_version is not None:
            context = ssl.create_default_context()
            context.minimum_version = ssl_version
            options["ssl_context"] = context
        return urllib3.PoolManager(**options)

    def _new_request_context(self) -> Optional[RequestContext]:
        if self.credentials_provider is None:
            return None
        credentials = self._coerce_credentials(
            self.credentials_provider.get_credentials()
        )
        return RequestContext(credentials)

    @staticmethod
    def _coerce_credentials(credentials) -> Credentials:
        if isinstance(credentials, Credentials):
            return credentials

        def read(attribute_name: str, getter_name: str, default=None):
            value = getattr(credentials, attribute_name, None)
            if value is not None:
                return value
            getter = getattr(credentials, getter_name, None)
            return getter() if callable(getter) else default

        access_key_id = read("access_key_id", "get_access_key_id")
        access_key_secret = read("access_key_secret", "get_access_key_secret")
        security_token = read("security_token", "get_security_token")
        if not access_key_id or not access_key_secret:
            raise OTSClientError(
                "credentials must provide access_key_id and access_key_secret."
            )
        return Credentials(access_key_id, access_key_secret, security_token)

    def _build_headers(
        self,
        action: str,
        body: bytes,
        request_context: Optional[RequestContext],
    ) -> Dict[str, str]:
        if self.api_key:
            headers = {
                "x-ots-instancename": self.instance_name,
                "x-ots-apikey": self.api_key,
                "User-Agent": self.user_agent,
            }
            headers.update(self.extra_headers)
        else:
            headers = build_request_headers(
                body=body,
                query="/" + action,
                instance_name=self.instance_name,
                request_context=request_context,
                signer=self.signer,
                user_agent=self.user_agent,
                extra_headers=self.extra_headers,
                now=self.clock(),
            )
        headers["Content-Type"] = "application/json"
        return headers

    def _send(
        self,
        action: str,
        body: bytes,
        headers: Mapping[str, str],
    ) -> Tuple[int, Dict[str, str], bytes]:
        try:
            response = self.transport.request(
                "POST",
                self.endpoint + "/" + action,
                body=body,
                headers=dict(headers),
                timeout=self.socket_timeout,
                retries=False,
                preload_content=True,
            )
        except Exception as error:
            raise OTSClientError("Failed to send request: %s" % error) from error
        response_headers = {
            str(key).lower(): str(value) for key, value in response.headers.items()
        }
        return response.status, response_headers, response.data

    def _validate_signed_response(
        self,
        action: str,
        status: int,
        headers: Mapping[str, str],
        body: bytes,
        request_context: RequestContext,
    ) -> None:
        if 200 <= status < 300:
            for name in (
                "x-ots-contentmd5",
                "x-ots-requestid",
                "x-ots-date",
                "x-ots-contenttype",
            ):
                if name not in headers:
                    raise OTSClientError('"%s" is missing in response header.' % name)

        expected_md5 = headers.get("x-ots-contentmd5")
        if expected_md5 and md5_base64(body) != expected_md5:
            raise OTSClientError("MD5 mismatch in response.")

        response_date = headers.get("x-ots-date")
        if response_date:
            try:
                server_time = datetime.strptime(
                    response_date, "%Y-%m-%dT%H:%M:%S.%fZ"
                ).replace(tzinfo=timezone.utc)
            except ValueError as error:
                raise OTSClientError("Invalid date format in response.") from error
            now = self.clock()
            if now.tzinfo is None:
                now = now.replace(tzinfo=timezone.utc)
            if (
                abs((server_time - now.astimezone(timezone.utc)).total_seconds())
                > MAX_RESPONSE_CLOCK_SKEW_SECONDS
            ):
                raise OTSClientError(
                    "The difference between date in response and system time is more than 15 minutes."
                )

        authorization = headers.get("authorization")
        if not authorization:
            if 200 <= status < 300:
                raise OTSClientError('"Authorization" is missing in response header.')
            return
        if not authorization.startswith("OTS "):
            raise OTSClientError("Invalid Authorization in response.")
        try:
            access_key_id, signature = authorization[4:].split(":", 1)
        except ValueError as error:
            raise OTSClientError("Invalid Authorization in response.") from error
        if access_key_id != request_context.credentials.access_key_id:
            raise OTSClientError("Invalid access key id in response.")
        expected_signature = self.signer.make_response_signature(
            "/" + action,
            headers,
            self.signer.get_signing_key(request_context),
        )
        if signature != expected_signature:
            raise OTSClientError("Invalid signature in response.")

    @staticmethod
    def _service_error(
        status: int,
        headers: Mapping[str, str],
        body: bytes,
    ) -> OTSServiceError:
        try:
            payload = json.loads(body.decode("utf-8"))
        except (UnicodeDecodeError, ValueError):
            payload = {}
        request_id = headers.get("x-ots-requestid") or payload.get("requestId", "")
        return OTSServiceError(
            status,
            payload.get("code", "HTTPError"),
            payload.get("message", "HTTP status %s" % status),
            request_id,
        )

    def _execute_with_retry(self, action: str, operation: Callable[[], Any]):
        retry_times = 0
        while True:
            try:
                return operation()
            except OTSServiceError as error:
                if not self.retry_policy.should_retry(retry_times, error, action):
                    raise
                delay = self.retry_policy.get_retry_delay(retry_times, error, action)
                self.sleep(delay)
                retry_times += 1

    def _request_once(self, action: str, body: bytes) -> Dict[str, Any]:
        request_context = self._new_request_context()
        headers = self._build_headers(action, body, request_context)
        status, response_headers, response_body = self._send(action, body, headers)
        if request_context is not None:
            self._validate_signed_response(
                action,
                status,
                response_headers,
                response_body,
                request_context,
            )
        if status < 200 or status >= 300:
            raise self._service_error(status, response_headers, response_body)
        try:
            return json.loads(response_body.decode(self.encoding))
        except (UnicodeDecodeError, ValueError) as error:
            raise OTSClientError("Failed to parse JSON response: %s" % error) from error

    def request(self, action: str, payload: Mapping[str, Any]) -> Dict[str, Any]:
        try:
            body = json.dumps(
                payload,
                ensure_ascii=False,
                separators=(",", ":"),
            ).encode(self.encoding)
        except (TypeError, ValueError) as error:
            raise OTSClientError("Failed to serialize request: %s" % error) from error
        return self._execute_with_retry(
            action, lambda: self._request_once(action, body)
        )
