package agentstorage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestClientErrorKindValues(t *testing.T) {
	tests := []struct {
		kind ClientErrorKind
		want string
	}{
		{ClientErrorKindRequestSerialize, "request_serialize"},
		{ClientErrorKindRequestBuild, "request_build"},
		{ClientErrorKindTransportContext, "transport_context"},
		{ClientErrorKindTransportTimeout, "transport_timeout"},
		{ClientErrorKindTransportNetwork, "transport_network"},
		{ClientErrorKindTransportOther, "transport_other"},
		{ClientErrorKindResponseRead, "response_read"},
		{ClientErrorKindResponseIntegrity, "response_integrity"},
		{ClientErrorKindResponseAuth, "response_auth"},
		{ClientErrorKindResponseDecode, "response_decode"},
		{ClientErrorKindResponseSemantic, "response_semantic"},
		{ClientErrorKindUnknown, "unknown"},
	}
	for _, test := range tests {
		if got := string(test.kind); got != test.want {
			t.Errorf("kind value = %q, want %q", got, test.want)
		}
	}
}

func TestHistoricalClientErrorKindIsUnknown(t *testing.T) {
	err := &ClientError{Message: "x"}
	if got := err.Kind(); got != ClientErrorKindUnknown {
		t.Fatalf("Kind() = %q, want %q", got, ClientErrorKindUnknown)
	}
	if got := err.Error(); got != "OTSClientError: x" {
		t.Fatalf("Error() = %q, want historical string", got)
	}

	var target *ClientError
	if !errors.As(err, &target) || target != err {
		t.Fatal("errors.As did not preserve *ClientError identity")
	}
	if errors.Unwrap(err) != nil {
		t.Fatal("historical ClientError unexpectedly has a cause")
	}
}

func TestClientErrorKindAndCauseAreNotRendered(t *testing.T) {
	cause := errors.New("sensitive-cause-marker")
	err := newClientErrorWithCause(ClientErrorKindTransportOther, cause, "safe message")

	if got := err.Kind(); got != ClientErrorKindTransportOther {
		t.Fatalf("Kind() = %q, want %q", got, ClientErrorKindTransportOther)
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is did not reach the typed cause")
	}
	for _, rendered := range []string{err.Error(), fmt.Sprint(err), fmt.Sprintf("%+v", err)} {
		if strings.Contains(rendered, string(ClientErrorKindTransportOther)) || strings.Contains(rendered, cause.Error()) {
			t.Fatal("default error rendering exposed kind or cause")
		}
	}
	encoded, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal("json.Marshal(ClientError) failed")
	}
	if strings.Contains(string(encoded), string(ClientErrorKindTransportOther)) || strings.Contains(string(encoded), cause.Error()) {
		t.Fatal("JSON rendering exposed kind or cause")
	}
}

func TestClientErrorInvalidInternalKindFallsBackToUnknown(t *testing.T) {
	err := newClientErrorWithCause(ClientErrorKind("future_value"), nil, "safe message")
	if got := err.Kind(); got != ClientErrorKindUnknown {
		t.Fatalf("Kind() = %q, want %q", got, ClientErrorKindUnknown)
	}
}
