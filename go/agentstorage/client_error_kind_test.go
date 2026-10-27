package agentstorage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type errorKindRoundTripFunc func(*http.Request) (*http.Response, error)

func (f errorKindRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errorKindReadCloser struct {
	err error
}

func (r *errorKindReadCloser) Read([]byte) (int, error) { return 0, r.err }
func (r *errorKindReadCloser) Close() error             { return nil }

type errorKindNetError struct {
	timeout bool
}

func (*errorKindNetError) Error() string   { return "typed network failure" }
func (e *errorKindNetError) Timeout() bool { return e.timeout }
func (*errorKindNetError) Temporary() bool { return false }

type errorKindMarshalPayload struct {
	err error
}

func (p errorKindMarshalPayload) MarshalJSON() ([]byte, error) {
	return nil, p.err
}

func newErrorKindTestClient(t *testing.T, endpoint string, transport http.RoundTripper) *Client {
	t.Helper()
	client, err := NewClient(Config{
		Endpoint:     endpoint,
		InstanceName: "instance",
		APIKey:       "test-api-key",
		HTTPClient:   &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal("NewClient failed")
	}
	return client
}

func requireClientErrorKind(t *testing.T, err error, want ClientErrorKind) *ClientError {
	t.Helper()
	if err == nil {
		t.Fatal("request unexpectedly succeeded")
	}
	var clientErr *ClientError
	if !errors.As(err, &clientErr) {
		t.Fatalf("error type = %T, want *ClientError", err)
	}
	if got := clientErr.Kind(); got != want {
		t.Fatalf("ClientError.Kind() = %q, want %q", got, want)
	}
	return clientErr
}

func errorKindResponse(status int, headers http.Header, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       body,
	}
}

func TestClientErrorKindRequestSerialize(t *testing.T) {
	cause := errors.New("marshal failure marker")
	client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("transport must not run after request serialization fails")
		return nil, nil
	}))

	err := client.Do(context.Background(), "Action", errorKindMarshalPayload{err: cause}, nil)
	requireClientErrorKind(t, err, ClientErrorKindRequestSerialize)
	if !errors.Is(err, cause) {
		t.Fatal("serialization cause was not preserved")
	}
	var marshalerErr *json.MarshalerError
	if !errors.As(err, &marshalerErr) {
		t.Fatal("serialization cause is not a *json.MarshalerError")
	}
}

func TestClientErrorKindRequestBuild(t *testing.T) {
	client := newErrorKindTestClient(t, "https://[::1", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("transport must not run after request building fails")
		return nil, nil
	}))

	err := client.Do(context.Background(), "Action", map[string]any{}, nil)
	requireClientErrorKind(t, err, ClientErrorKindRequestBuild)
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatal("request build cause is not a *url.Error")
	}
}

func TestClientErrorKindsForTransportFailures(t *testing.T) {
	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, req.Context().Err()
		}))

		err := client.Do(ctx, "Action", map[string]any{}, nil)
		requireClientErrorKind(t, err, ClientErrorKindTransportContext)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("context cancellation cause was not preserved")
		}
	})

	t.Run("context deadline", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, req.Context().Err()
		}))

		err := client.Do(ctx, "Action", map[string]any{}, nil)
		requireClientErrorKind(t, err, ClientErrorKindTransportTimeout)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("context deadline cause was not preserved")
		}
	})

	t.Run("net timeout", func(t *testing.T) {
		cause := &errorKindNetError{timeout: true}
		client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, cause
		}))

		err := client.Do(context.Background(), "Action", map[string]any{}, nil)
		requireClientErrorKind(t, err, ClientErrorKindTransportTimeout)
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Fatal("timeout cause is not available as net.Error")
		}
	})

	t.Run("url network", func(t *testing.T) {
		cause := &url.Error{Err: errors.New("network failure marker")}
		client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, cause
		}))

		err := client.Do(context.Background(), "Action", map[string]any{}, nil)
		requireClientErrorKind(t, err, ClientErrorKindTransportNetwork)
		var urlErr *url.Error
		if !errors.As(err, &urlErr) {
			t.Fatal("network cause is not available as *url.Error")
		}
	})

	t.Run("net network", func(t *testing.T) {
		cause := &errorKindNetError{}
		client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, cause
		}))

		err := client.Do(context.Background(), "Action", map[string]any{}, nil)
		requireClientErrorKind(t, err, ClientErrorKindTransportNetwork)
		var netErr *errorKindNetError
		if !errors.As(err, &netErr) {
			t.Fatal("network cause is not available as its net.Error type")
		}
	})

	t.Run("other", func(t *testing.T) {
		cause := errors.New("other failure marker")
		client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, cause
		}))

		err := client.Do(context.Background(), "Action", map[string]any{}, nil)
		requireClientErrorKind(t, err, ClientErrorKindTransportOther)
		if !errors.Is(err, cause) {
			t.Fatal("other transport cause was not preserved")
		}
	})
}

func TestClientErrorKindsForResponseBodyReadFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		kind ClientErrorKind
	}{
		{name: "context canceled", err: context.Canceled, kind: ClientErrorKindTransportContext},
		{name: "context deadline", err: context.DeadlineExceeded, kind: ClientErrorKindTransportTimeout},
		{name: "net timeout", err: &errorKindNetError{timeout: true}, kind: ClientErrorKindTransportTimeout},
		{name: "ordinary read failure", err: errors.New("read failure marker"), kind: ClientErrorKindResponseRead},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return errorKindResponse(http.StatusOK, make(http.Header), &errorKindReadCloser{err: test.err}), nil
			}))

			err := client.Do(context.Background(), "Action", map[string]any{}, nil)
			requireClientErrorKind(t, err, test.kind)
			if !errors.Is(err, test.err) {
				t.Fatal("response read cause was not preserved")
			}
		})
	}
}

func validSignedResponseFixture(body []byte) (*otsJSONClient, map[string]string, string) {
	const query = "/Action"
	client := &otsJSONClient{
		accessKeyID:     "ak",
		accessKeySecret: "sk",
		signer:          signerV2{},
	}
	headers := map[string]string{
		"x-ots-contentmd5":  md5Base64(body),
		"x-ots-requestid":   "request-id",
		"x-ots-date":        "date",
		"x-ots-contenttype": "application/json",
	}
	signature := responseSignature(query, headers, []byte(client.accessKeySecret))
	headers["authorization"] = "OTS " + client.accessKeyID + ":" + signature
	return client, headers, query
}

func cloneStringMap(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func TestClientErrorKindResponseIntegrity(t *testing.T) {
	body := []byte(`{}`)
	client, validHeaders, query := validSignedResponseFixture(body)

	for _, missing := range []string{"x-ots-contentmd5", "x-ots-requestid", "x-ots-date", "x-ots-contenttype"} {
		t.Run("missing "+missing, func(t *testing.T) {
			headers := cloneStringMap(validHeaders)
			delete(headers, missing)
			err := client.validateResponse(headers, body, http.StatusOK, query, "")
			requireClientErrorKind(t, err, ClientErrorKindResponseIntegrity)
		})
	}

	t.Run("md5 mismatch", func(t *testing.T) {
		headers := cloneStringMap(validHeaders)
		headers["x-ots-contentmd5"] = "mismatch"
		err := client.validateResponse(headers, body, http.StatusOK, query, "")
		requireClientErrorKind(t, err, ClientErrorKindResponseIntegrity)
	})
}

func TestClientErrorKindResponseAuth(t *testing.T) {
	body := []byte(`{}`)
	client, validHeaders, query := validSignedResponseFixture(body)
	tests := []struct {
		name          string
		authorization string
		remove        bool
	}{
		{name: "missing", remove: true},
		{name: "scheme", authorization: "Bearer opaque"},
		{name: "access key", authorization: "OTS other:opaque"},
		{name: "signature", authorization: "OTS ak:invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers := cloneStringMap(validHeaders)
			if test.remove {
				delete(headers, "authorization")
			} else {
				headers["authorization"] = test.authorization
			}
			err := client.validateResponse(headers, body, http.StatusOK, query, "")
			requireClientErrorKind(t, err, ClientErrorKindResponseAuth)
		})
	}
}

func TestClientErrorKindResponseDecode(t *testing.T) {
	client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return errorKindResponse(http.StatusOK, make(http.Header), io.NopCloser(errorReaderString("{"))), nil
	}))
	var result map[string]any
	err := client.Do(context.Background(), "Action", map[string]any{}, &result)
	requireClientErrorKind(t, err, ClientErrorKindResponseDecode)
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatal("decode cause is not available as *json.SyntaxError")
	}
}

type errorReaderString string

func (s errorReaderString) Read(p []byte) (int, error) {
	if len(s) == 0 {
		return 0, io.EOF
	}
	return copy(p, s), io.EOF
}

func TestResponseSemanticKindIsReservedAndEmptyObjectStaysValid(t *testing.T) {
	client := newErrorKindTestClient(t, "https://example.invalid", errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return errorKindResponse(http.StatusOK, make(http.Header), io.NopCloser(errorReaderString(`{}`))), nil
	}))
	var result map[string]any
	if err := client.Do(context.Background(), "Action", map[string]any{}, &result); err != nil {
		t.Fatal("empty JSON object response unexpectedly failed")
	}
	if result == nil {
		t.Fatal("empty JSON object did not decode")
	}
}

func TestSigned4xxResponseValidationPrecedesServiceError(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		kind    ClientErrorKind
	}{
		{name: "md5 mismatch", headers: http.Header{"x-ots-contentmd5": []string{"mismatch"}}, kind: ClientErrorKindResponseIntegrity},
		{name: "invalid authorization", headers: http.Header{"Authorization": []string{"Bearer opaque"}}, kind: ClientErrorKindResponseAuth},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient(Config{
				Endpoint:        "https://example.invalid",
				InstanceName:    "instance",
				AccessKeyID:     "ak",
				AccessKeySecret: "sk",
				HTTPClient: &http.Client{Transport: errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
					return errorKindResponse(http.StatusBadRequest, test.headers, io.NopCloser(errorReaderString(`{"code":"BAD_REQUEST"}`))), nil
				})},
			})
			if err != nil {
				t.Fatal("NewClient failed")
			}

			err = client.Do(context.Background(), "Action", map[string]any{}, nil)
			requireClientErrorKind(t, err, test.kind)
			var serviceErr *ServiceError
			if errors.As(err, &serviceErr) {
				t.Fatal("unverified signed 4xx response was also classified as ServiceError")
			}
		})
	}
}

func TestSigned4xxWithoutValidationHeadersRemainsServiceError(t *testing.T) {
	client, err := NewClient(Config{
		Endpoint:        "https://example.invalid",
		InstanceName:    "instance",
		AccessKeyID:     "ak",
		AccessKeySecret: "sk",
		HTTPClient: &http.Client{Transport: errorKindRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return errorKindResponse(http.StatusBadRequest, make(http.Header), io.NopCloser(errorReaderString(`{"code":"BAD_REQUEST"}`))), nil
		})},
	})
	if err != nil {
		t.Fatal("NewClient failed")
	}

	err = client.Do(context.Background(), "Action", map[string]any{}, nil)
	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) || serviceErr.StatusCode != http.StatusBadRequest || serviceErr.Code != "BAD_REQUEST" {
		t.Fatalf("err=%v, want BAD_REQUEST ServiceError", err)
	}
}

func TestResponseSigningKeyFailureIsResponseAuth(t *testing.T) {
	client := &otsJSONClient{
		accessKeyID:     "ak",
		accessKeySecret: "sk",
		signer:          &signerV4{region: "test-region"},
	}
	err := client.validateResponse(
		map[string]string{"authorization": "OTS ak:signature"},
		[]byte(`{"code":"BAD_REQUEST"}`),
		http.StatusBadRequest,
		"/Action",
		"",
	)
	requireClientErrorKind(t, err, ClientErrorKindResponseAuth)
	if !strings.Contains(err.Error(), "v4 sign_date is None") {
		t.Fatalf("signing-key cause was not preserved: %v", err)
	}
}

func TestUnclassifiedSDKClientErrorKindIsUnknown(t *testing.T) {
	_, err := NewClient(Config{})
	requireClientErrorKind(t, err, ClientErrorKindUnknown)
}
