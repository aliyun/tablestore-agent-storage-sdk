"""Retry policies compatible with the former Tablestore Python client."""

import math
import random

from tablestore_agent_storage.errors import OTSServiceError


class RetryPolicy:
    """Interface for caller-provided retry strategies."""

    def should_retry(
        self, retry_times: int, exception: OTSServiceError, api_name: str
    ) -> bool:
        raise NotImplementedError()

    def get_retry_delay(
        self, retry_times: int, exception: OTSServiceError, api_name: str
    ) -> float:
        raise NotImplementedError()


class RetryUtil:
    _ALWAYS_RETRY_CODES = {
        "OTSRowOperationConflict",
        "OTSNotEnoughCapacityUnit",
        "OTSTableNotReady",
        "OTSPartitionUnavailable",
        "OTSServerBusy",
        "OTSOperationConflict",
        "OTSOperationThrottled",
    }
    _STABILITY_CODES = {
        "OTSTimeout",
        "OTSInternalServerError",
        "OTSServerUnavailable",
    }
    _REPEATABLE_APIS = {
        "ListTable",
        "DescribeTable",
        "GetRow",
        "BatchGetRow",
        "GetRange",
        "GetTimeseriesData",
        "ListTimeseriesTable",
        "DescribeTimeseriesTable",
        "QueryTimeseriesMeta",
    }

    @classmethod
    def should_retry_no_matter_which_api(cls, exception: OTSServiceError) -> bool:
        if exception.code in cls._ALWAYS_RETRY_CODES:
            return True
        return (
            exception.code == "OTSQuotaExhausted"
            and exception.message == "Too frequent table operations."
        )

    @classmethod
    def is_repeatable_api(cls, api_name: str) -> bool:
        return api_name in cls._REPEATABLE_APIS

    @classmethod
    def should_retry_when_api_repeatable(
        cls,
        retry_times: int,
        exception: OTSServiceError,
        api_name: str,
    ) -> bool:
        del retry_times, api_name
        return exception.code in cls._STABILITY_CODES or exception.http_status in {
            500,
            502,
            503,
        }

    @classmethod
    def is_server_throttling_exception(cls, exception: OTSServiceError) -> bool:
        if exception.code in {
            "OTSServerBusy",
            "OTSNotEnoughCapacityUnit",
            "OTSOperationThrottled",
        }:
            return True
        return (
            exception.code == "OTSQuotaExhausted"
            and exception.message == "Too frequent table operations."
        )


class DefaultRetryPolicy(RetryPolicy):
    """The former SDK's bounded exponential backoff policy."""

    max_retry_times = 20
    max_retry_delay = 3
    scale_factor = 2
    server_throttling_exception_delay_factor = 0.5
    stability_exception_delay_factor = 0.2

    def is_repeatable_api(self, api_name: str) -> bool:
        return RetryUtil.is_repeatable_api(api_name)

    def _can_retry(
        self, retry_times: int, exception: OTSServiceError, api_name: str
    ) -> bool:
        if RetryUtil.should_retry_no_matter_which_api(exception):
            return True
        return self.is_repeatable_api(
            api_name
        ) and RetryUtil.should_retry_when_api_repeatable(
            retry_times,
            exception,
            api_name,
        )

    def should_retry(
        self, retry_times: int, exception: OTSServiceError, api_name: str
    ) -> bool:
        return retry_times < self.max_retry_times and self._can_retry(
            retry_times, exception, api_name
        )

    def get_retry_delay(
        self, retry_times: int, exception: OTSServiceError, api_name: str
    ) -> float:
        del api_name
        if RetryUtil.is_server_throttling_exception(exception):
            delay_factor = self.server_throttling_exception_delay_factor
        else:
            delay_factor = self.stability_exception_delay_factor
        delay_limit = min(
            delay_factor * math.pow(self.scale_factor, retry_times),
            self.max_retry_delay,
        )
        return delay_limit * 0.5 + delay_limit * 0.5 * random.random()


class NoRetryPolicy(RetryPolicy):
    def should_retry(
        self, retry_times: int, exception: OTSServiceError, api_name: str
    ) -> bool:
        del retry_times, exception, api_name
        return False

    def get_retry_delay(
        self, retry_times: int, exception: OTSServiceError, api_name: str
    ) -> float:
        del retry_times, exception, api_name
        return 0


class NoDelayRetryPolicy(DefaultRetryPolicy):
    def get_retry_delay(
        self, retry_times: int, exception: OTSServiceError, api_name: str
    ) -> float:
        del retry_times, exception, api_name
        return 0


class WriteRetryPolicy(DefaultRetryPolicy):
    _WRITE_APIS = {
        "ListTable",
        "DescribeTable",
        "GetRow",
        "BatchGetRow",
        "GetRange",
        "PutRow",
        "UpdateRow",
        "DeleteRow",
        "BatchWriteRow",
    }

    def is_repeatable_api(self, api_name: str) -> bool:
        return api_name in self._WRITE_APIS
