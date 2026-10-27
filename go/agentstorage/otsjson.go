package agentstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// otsJSONClient speaks the OTS HTTP JSON protocol: POST {endpoint}/{Action}
// with a JSON body, authenticated either by request signing (v2/v4) or by an
// API key.
type otsJSONClient struct {
	endpoint        string
	instanceName    string
	accessKeyID     string
	accessKeySecret string
	securityToken   string
	apiKey          string
	signer          signer
	httpClient      *http.Client
	userAgent       string
	extraHeaders    map[string]string
	// clock overrides the x-ots-date value in tests; nil means real time.
	clock func() time.Time
}

func (c *otsJSONClient) request(ctx context.Context, action string, payload any) ([]byte, error) {
	status, respHeaders, respBody, signDate, err := c.send(ctx, action, payload)
	if err != nil {
		return nil, err
	}

	if c.apiKey == "" {
		if err := c.validateResponse(respHeaders, respBody, status, "/"+action, signDate); err != nil {
			return nil, err
		}
	}

	if status < 200 || status >= 300 {
		return nil, buildServiceError(status, respHeaders, respBody)
	}
	return respBody, nil
}

func (c *otsJSONClient) send(ctx context.Context, action string, payload any) (int, map[string]string, []byte, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, nil, "", newClientErrorWithCause(ClientErrorKindRequestSerialize, err, "failed to serialize request: %v", err)
	}

	var headers map[string]string
	var signDate string
	if c.apiKey != "" {
		headers = map[string]string{
			"x-ots-instancename": c.instanceName,
			"x-ots-apikey":       c.apiKey,
			"User-Agent":         c.userAgent,
		}
		for key, value := range c.extraHeaders {
			headers[key] = value
		}
	} else {
		var date time.Time
		if c.clock != nil {
			date = c.clock()
		}
		headers, signDate, err = buildRequestHeaders(requestHeaderInput{
			body:            body,
			query:           "/" + action,
			instanceName:    c.instanceName,
			accessKeyID:     c.accessKeyID,
			accessKeySecret: c.accessKeySecret,
			securityToken:   c.securityToken,
			signer:          c.signer,
			userAgent:       c.userAgent,
			extraHeaders:    c.extraHeaders,
			date:            date,
		})
		if err != nil {
			return 0, nil, nil, "", err
		}
	}
	headers["Content-Type"] = "application/json"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/"+action, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, "", newClientErrorWithCause(ClientErrorKindRequestBuild, err, "failed to build request: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, nil, "", newClientErrorWithCause(classifyTransportError(err), err, "failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, "", newClientErrorWithCause(classifyResponseReadError(err), err, "failed to read response: %v", err)
	}

	return resp.StatusCode, normalizeHeaders(resp.Header), respBody, signDate, nil
}

func classifyResponseReadError(err error) ClientErrorKind {
	if errors.Is(err, context.Canceled) {
		return ClientErrorKindTransportContext
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ClientErrorKindTransportTimeout
	}
	var timeoutErr net.Error
	if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
		return ClientErrorKindTransportTimeout
	}
	return ClientErrorKindResponseRead
}

func classifyTransportError(err error) ClientErrorKind {
	if errors.Is(err, context.Canceled) {
		return ClientErrorKindTransportContext
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ClientErrorKindTransportTimeout
	}
	var timeoutErr net.Error
	if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
		return ClientErrorKindTransportTimeout
	}

	// http.Client.Do wraps every RoundTripper error in a *url.Error. Inspect
	// the wrapped cause so a bare non-network error remains distinguishable
	// from a URL or net error returned by the transport.
	var outerURLError *url.Error
	if errors.As(err, &outerURLError) {
		var innerURLError *url.Error
		if errors.As(outerURLError.Err, &innerURLError) {
			return ClientErrorKindTransportNetwork
		}
		var innerNetError net.Error
		if errors.As(outerURLError.Err, &innerNetError) {
			return ClientErrorKindTransportNetwork
		}
		return ClientErrorKindTransportOther
	}

	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return ClientErrorKindTransportNetwork
	}
	return ClientErrorKindTransportOther
}

func normalizeHeaders(h http.Header) map[string]string {
	normalized := make(map[string]string, len(h))
	for key, values := range h {
		if len(values) > 0 {
			normalized[strings.ToLower(key)] = values[0]
		}
	}
	return normalized
}

func buildServiceError(status int, headers map[string]string, body []byte) *ServiceError {
	var payload struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	}
	_ = json.Unmarshal(body, &payload)
	requestID := headers["x-ots-requestid"]
	if requestID == "" {
		requestID = payload.RequestID
	}
	message := payload.Message
	if message == "" {
		message = http.StatusText(status)
	}
	return &ServiceError{
		StatusCode: status,
		Code:       payload.Code,
		Message:    message,
		RequestID:  requestID,
	}
}

func (c *otsJSONClient) validateResponse(headers map[string]string, body []byte, status int, query, signDate string) error {
	if status >= 200 && status < 300 {
		for _, header := range []string{"x-ots-contentmd5", "x-ots-requestid", "x-ots-date", "x-ots-contenttype"} {
			if _, ok := headers[header]; !ok {
				return newClientErrorWithCause(ClientErrorKindResponseIntegrity, nil, "%q is missing in response header.", header)
			}
		}
	}

	if expected := headers["x-ots-contentmd5"]; expected != "" && md5Base64(body) != expected {
		return newClientErrorWithCause(ClientErrorKindResponseIntegrity, nil, "MD5 mismatch in response.")
	}

	authorization := headers["authorization"]
	if authorization == "" {
		if status >= 200 && status < 300 {
			return newClientErrorWithCause(ClientErrorKindResponseAuth, nil, "\"Authorization\" is missing in response header.")
		}
		return nil
	}

	if !strings.HasPrefix(authorization, "OTS ") {
		return newClientErrorWithCause(ClientErrorKindResponseAuth, nil, "Invalid Authorization in response.")
	}
	accessKeyID, signature, found := strings.Cut(authorization[4:], ":")
	if !found || accessKeyID != c.accessKeyID {
		return newClientErrorWithCause(ClientErrorKindResponseAuth, nil, "Invalid access key id in response.")
	}

	signingKey, err := c.signer.signingKey(c.accessKeySecret, signDate)
	if err != nil {
		return newClientErrorWithCause(ClientErrorKindResponseAuth, err, "failed to derive response signing key: %v", err)
	}
	if signature != responseSignature(query, headers, signingKey) {
		return newClientErrorWithCause(ClientErrorKindResponseAuth, nil, "Invalid signature in response.")
	}
	return nil
}
