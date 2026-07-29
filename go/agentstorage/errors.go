package agentstorage

import "fmt"

// ClientError is returned for client-side failures: invalid configuration,
// request building/serialization problems, transport errors and invalid
// responses (missing headers, MD5/signature mismatch).
type ClientError struct {
	Message string
}

func (e *ClientError) Error() string {
	return "OTSClientError: " + e.Message
}

func newClientError(format string, args ...any) *ClientError {
	return &ClientError{Message: fmt.Sprintf(format, args...)}
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
