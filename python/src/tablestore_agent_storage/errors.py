"""Exceptions raised by the internal OTS JSON client."""

from typing import Optional


class OTSError(Exception):
    """Base class for OTS client and service failures."""


class OTSClientError(OTSError):
    """A client-side configuration, transport, or response failure."""

    def __init__(self, message: str, http_status: Optional[int] = None):
        super().__init__(message)
        self.message = message
        self.http_status = http_status

    def get_http_status(self) -> Optional[int]:
        return self.http_status

    def get_error_message(self) -> str:
        return self.message

    def __str__(self) -> str:
        return self.message


class OTSServiceError(OTSError):
    """An error response returned by the OTS service."""

    def __init__(self, http_status: int, code: str, message: str, request_id: str = ""):
        super().__init__(message)
        self.http_status = http_status
        self.code = code
        self.message = message
        self.request_id = request_id

    def get_http_status(self) -> int:
        return self.http_status

    def get_error_code(self) -> str:
        return self.code

    def get_error_message(self) -> str:
        return self.message

    def get_request_id(self) -> str:
        return self.request_id

    def __str__(self) -> str:
        return "ErrorCode: %s, ErrorMessage: %s, RequestID: %s" % (
            self.code,
            self.message,
            self.request_id,
        )
