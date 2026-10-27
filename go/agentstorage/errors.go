package agentstorage

import "fmt"

// ClientErrorKind identifies the SDK boundary where a client-side failure
// occurred. It is safe to use for programmatic error handling.
type ClientErrorKind string

const (
	ClientErrorKindRequestSerialize  ClientErrorKind = "request_serialize"
	ClientErrorKindRequestBuild      ClientErrorKind = "request_build"
	ClientErrorKindTransportContext  ClientErrorKind = "transport_context"
	ClientErrorKindTransportTimeout  ClientErrorKind = "transport_timeout"
	ClientErrorKindTransportNetwork  ClientErrorKind = "transport_network"
	ClientErrorKindTransportOther    ClientErrorKind = "transport_other"
	ClientErrorKindResponseRead      ClientErrorKind = "response_read"
	ClientErrorKindResponseIntegrity ClientErrorKind = "response_integrity"
	ClientErrorKindResponseAuth      ClientErrorKind = "response_auth"
	ClientErrorKindResponseDecode    ClientErrorKind = "response_decode"
	ClientErrorKindResponseSemantic  ClientErrorKind = "response_semantic"
	ClientErrorKindUnknown           ClientErrorKind = "unknown"
)

// ClientError is returned for client-side failures: invalid configuration,
// request building/serialization problems, transport errors and invalid
// responses (missing headers, MD5/signature mismatch).
type ClientError struct {
	Message string

	kind  ClientErrorKind
	cause error
}

func (e *ClientError) Error() string {
	return "OTSClientError: " + e.Message
}

// Kind returns the stable category of this client-side failure. ClientError
// values created without SDK metadata safely report ClientErrorKindUnknown.
func (e *ClientError) Kind() ClientErrorKind {
	if e == nil {
		return ClientErrorKindUnknown
	}
	switch e.kind {
	case ClientErrorKindRequestSerialize,
		ClientErrorKindRequestBuild,
		ClientErrorKindTransportContext,
		ClientErrorKindTransportTimeout,
		ClientErrorKindTransportNetwork,
		ClientErrorKindTransportOther,
		ClientErrorKindResponseRead,
		ClientErrorKindResponseIntegrity,
		ClientErrorKindResponseAuth,
		ClientErrorKindResponseDecode,
		ClientErrorKindResponseSemantic,
		ClientErrorKindUnknown:
		return e.kind
	default:
		return ClientErrorKindUnknown
	}
}

// Unwrap returns the typed underlying cause, when one exists.
func (e *ClientError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newClientError(format string, args ...any) *ClientError {
	return &ClientError{Message: fmt.Sprintf(format, args...)}
}

func newClientErrorWithCause(kind ClientErrorKind, cause error, format string, args ...any) *ClientError {
	return &ClientError{
		Message: fmt.Sprintf(format, args...),
		kind:    kind,
		cause:   cause,
	}
}

// ServiceError is returned when the service answers with a non-2xx status.
type ServiceError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("OTSServiceError: status=%d code=%s message=%s requestId=%s",
		e.StatusCode, e.Code, e.Message, e.RequestID)
}
