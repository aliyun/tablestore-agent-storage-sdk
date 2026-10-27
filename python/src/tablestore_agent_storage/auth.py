"""OTS request signing primitives used by the internal JSON client."""

import base64
import hashlib
import hmac
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Callable, Dict, Mapping, Optional, Tuple, Union
from urllib.parse import parse_qsl, urlencode, urlsplit

from tablestore_agent_storage.errors import OTSClientError

API_VERSION = "2015-12-31"
OTS_HEADER_PREFIX = "x-ots-"
V4_SIGNATURE_PRODUCT = "ots"
V4_SIGNATURE_PREFIX = "aliyun_v4"
V4_SIGNATURE_CONSTANT = "aliyun_v4_request"
V4_SIGNATURE_SALT = "ots"


@dataclass(frozen=True)
class Credentials:
    access_key_id: str
    access_key_secret: str
    security_token: Optional[str] = None

    def get_access_key_id(self) -> str:
        return self.access_key_id

    def get_access_key_secret(self) -> str:
        return self.access_key_secret

    def get_security_token(self) -> Optional[str]:
        return self.security_token


class StaticCredentialsProvider:
    def __init__(self, credentials: Credentials):
        self._credentials = credentials

    def get_credentials(self) -> Credentials:
        return self._credentials


class RequestContext:
    def __init__(self, credentials: Credentials, sign_date: Optional[str] = None):
        self.credentials = credentials
        self.sign_date = sign_date

    def get_credentials(self) -> Credentials:
        return self.credentials

    def set_sign_date(self, sign_date: str) -> None:
        self.sign_date = sign_date

    def get_sign_date(self) -> Optional[str]:
        return self.sign_date


def _as_bytes(value: Union[str, bytes], encoding: str = "utf-8") -> bytes:
    return value.encode(encoding) if isinstance(value, str) else value


def _hmac_bytes(
    key: Union[str, bytes],
    content: str,
    digest,
    encoding: str = "utf-8",
) -> bytes:
    return hmac.new(
        _as_bytes(key, encoding),
        content.encode(encoding),
        digest,
    ).digest()


def hmac_sha1_base64(
    key: Union[str, bytes], content: str, encoding: str = "utf-8"
) -> str:
    return base64.b64encode(_hmac_bytes(key, content, hashlib.sha1, encoding)).decode(
        encoding
    )


def hmac_sha256_base64(
    key: Union[str, bytes], content: str, encoding: str = "utf-8"
) -> str:
    return base64.b64encode(_hmac_bytes(key, content, hashlib.sha256, encoding)).decode(
        encoding
    )


def _hmac_sha256_bytes(
    key: Union[str, bytes], content: str, encoding: str = "utf-8"
) -> bytes:
    return _hmac_bytes(key, content, hashlib.sha256, encoding)


def md5_base64(content: bytes) -> str:
    return base64.b64encode(hashlib.md5(content).digest()).decode("ascii")


def sort_and_encode_query(query_string: str) -> str:
    if not query_string:
        return ""
    return urlencode(sorted(parse_qsl(query_string, keep_blank_values=True)))


def _split_query(query: str) -> Tuple[str, str]:
    parsed = urlsplit(query)
    return parsed.path, parsed.query


def make_headers_string(headers: Mapping[str, str]) -> str:
    items = [
        "%s:%s" % (key.lower(), str(value).strip())
        for key, value in headers.items()
        if key.lower().startswith(OTS_HEADER_PREFIX)
    ]
    return "\n".join(sorted(items))


def request_signature_string(query: str, headers: Mapping[str, str]) -> str:
    uri, raw_query = _split_query(query)
    return "%s\nPOST\n%s\n%s\n" % (
        uri,
        sort_and_encode_query(raw_query),
        make_headers_string(headers),
    )


class SignBase:
    def __init__(self, encoding: str = "utf-8"):
        self.encoding = encoding

    def get_signing_key(self, request_context: RequestContext) -> bytes:
        raise NotImplementedError()

    def add_request_signature(
        self,
        query: str,
        headers: Dict[str, str],
        request_context: RequestContext,
    ) -> None:
        raise NotImplementedError()

    def make_response_signature(
        self,
        query: str,
        headers: Mapping[str, str],
        signing_key: Union[str, bytes],
    ) -> str:
        uri, _ = _split_query(query)
        return hmac_sha1_base64(
            signing_key,
            "%s\n%s" % (make_headers_string(headers), uri),
            self.encoding,
        )


class SignV2(SignBase):
    def get_signing_key(self, request_context: RequestContext) -> bytes:
        return request_context.credentials.access_key_secret.encode(self.encoding)

    def add_request_signature(
        self,
        query: str,
        headers: Dict[str, str],
        request_context: RequestContext,
    ) -> None:
        headers["x-ots-signature"] = hmac_sha1_base64(
            self.get_signing_key(request_context),
            request_signature_string(query, headers),
            self.encoding,
        )


class SignV4(SignBase):
    def __init__(
        self,
        region: str,
        sign_date: Optional[str] = None,
        auto_update_v4_sign: Optional[bool] = None,
        clock: Optional[Callable[[], datetime]] = None,
        encoding: str = "utf-8",
    ):
        super().__init__(encoding)
        if not isinstance(region, str) or not region:
            raise OTSClientError("region is not str or is empty.")
        self.region = region
        self.sign_date = sign_date
        self.auto_update_v4_sign = (
            sign_date is None if auto_update_v4_sign is None else auto_update_v4_sign
        )
        if sign_date is None:
            self.auto_update_v4_sign = True
        self.clock = clock or (lambda: datetime.now(timezone.utc))

    def _current_sign_date(self) -> str:
        if self.auto_update_v4_sign:
            return self.clock().astimezone(timezone.utc).strftime("%Y%m%d")
        if self.sign_date is None:
            raise OTSClientError("v4 sign_date is None.")
        return self.sign_date

    def get_signing_key(self, request_context: RequestContext) -> bytes:
        sign_date = request_context.get_sign_date()
        if not sign_date:
            raise OTSClientError("v4 sign_date is None.")
        secret = request_context.credentials.access_key_secret
        first = _hmac_sha256_bytes(
            V4_SIGNATURE_PREFIX + secret, sign_date, self.encoding
        )
        second = _hmac_sha256_bytes(first, self.region, self.encoding)
        third = _hmac_sha256_bytes(second, V4_SIGNATURE_PRODUCT, self.encoding)
        fourth = _hmac_sha256_bytes(third, V4_SIGNATURE_CONSTANT, self.encoding)
        return base64.b64encode(fourth)

    def add_request_signature(
        self,
        query: str,
        headers: Dict[str, str],
        request_context: RequestContext,
    ) -> None:
        sign_date = self._current_sign_date()
        request_context.set_sign_date(sign_date)
        headers["x-ots-signdate"] = sign_date
        headers["x-ots-signregion"] = self.region
        headers["x-ots-signaturev4"] = hmac_sha256_base64(
            self.get_signing_key(request_context),
            request_signature_string(query, headers) + V4_SIGNATURE_SALT,
            self.encoding,
        )


def build_date_header(now: datetime) -> str:
    if now.tzinfo is None:
        now = now.replace(tzinfo=timezone.utc)
    return now.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.000Z")


def build_request_headers(
    body: bytes,
    query: str,
    instance_name: str,
    request_context: RequestContext,
    signer: SignBase,
    user_agent: str,
    extra_headers: Optional[Mapping[str, str]] = None,
    now: Optional[datetime] = None,
) -> Dict[str, str]:
    credentials = request_context.credentials
    headers = {
        "x-ots-date": build_date_header(now or datetime.now(timezone.utc)),
        "x-ots-apiversion": API_VERSION,
        "x-ots-accesskeyid": credentials.access_key_id,
        "x-ots-instancename": instance_name,
        "x-ots-contentmd5": md5_base64(body),
    }
    if credentials.security_token:
        headers["x-ots-ststoken"] = credentials.security_token
    if extra_headers:
        headers.update(extra_headers)
    signer.add_request_signature(query, headers, request_context)
    headers["User-Agent"] = user_agent
    return headers
