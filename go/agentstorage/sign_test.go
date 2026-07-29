package agentstorage

import (
	"testing"
	"time"
)

// Fixtures shared with the Python and TypeScript SDK test suites.
const (
	testAkSecret = "test_key"
	testRegion   = "test-region"
	testSignDate = "20250410"
	testQuery    = "test_query"
)

func testHeaders() map[string]string {
	return map[string]string{"x-ots-test": "test"}
}

func TestSignV2Fixtures(t *testing.T) {
	headers := testHeaders()
	s := signerV2{}
	if _, err := s.addSignature(testQuery, headers, testAkSecret); err != nil {
		t.Fatal(err)
	}
	if got, want := headers["x-ots-signature"], "QDhzLv7VESBJtYQY4Li0IhSUOdg="; got != want {
		t.Errorf("request signature = %q, want %q", got, want)
	}
	key, err := s.signingKey(testAkSecret, "")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := responseSignature(testQuery, testHeaders(), key), "UjJK/SWed0n9o6JYxvApHGaQABo="; got != want {
		t.Errorf("response signature = %q, want %q", got, want)
	}
}

func TestSignV4Fixtures(t *testing.T) {
	headers := testHeaders()
	s := &signerV4{region: testRegion, signDate: testSignDate}
	signDate, err := s.addSignature(testQuery, headers, testAkSecret)
	if err != nil {
		t.Fatal(err)
	}
	if signDate != testSignDate {
		t.Errorf("signDate = %q, want %q", signDate, testSignDate)
	}
	if got := headers["x-ots-signregion"]; got != testRegion {
		t.Errorf("x-ots-signregion = %q, want %q", got, testRegion)
	}
	if got := headers["x-ots-signdate"]; got != testSignDate {
		t.Errorf("x-ots-signdate = %q, want %q", got, testSignDate)
	}
	if got, want := headers["x-ots-signaturev4"], "yXnOpODWaU1EYAlLP3l25ksj010uGHS7uxIt5Qiwz4o="; got != want {
		t.Errorf("request signature = %q, want %q", got, want)
	}
	key, err := s.signingKey(testAkSecret, testSignDate)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(key), "nToxlXrxgCm0L5J0nr/qq/GmtgN9GVBhiRLzdLaVUP0="; got != want {
		t.Errorf("signing key = %q, want %q", got, want)
	}
	if got, want := responseSignature(testQuery, testHeaders(), key), "vIhaUGwv/JSg8ctLNyxbNeNv69A="; got != want {
		t.Errorf("response signature = %q, want %q", got, want)
	}
}

func TestBuildDateHeader(t *testing.T) {
	date := time.Date(2025, 4, 10, 1, 2, 3, 456_000_000, time.UTC)
	if got, want := buildDateHeader(date), "2025-04-10T01:02:03.000Z"; got != want {
		t.Errorf("buildDateHeader = %q, want %q", got, want)
	}
}

func TestBuildRequestHeaders(t *testing.T) {
	headers, _, err := buildRequestHeaders(requestHeaderInput{
		body:            []byte("hello"),
		query:           "/CreateKnowledgeBase",
		instanceName:    "instance",
		accessKeyID:     "ak",
		accessKeySecret: "sk",
		signer:          signerV2{},
		userAgent:       "test-agent",
		date:            time.Date(2025, 4, 10, 1, 2, 3, 456_000_000, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	expectations := map[string]string{
		"x-ots-apiversion":   "2015-12-31",
		"x-ots-instancename": "instance",
		"User-Agent":         "test-agent",
		"x-ots-contentmd5":   "XUFAKrxLKna5cZ2REBfFkg==",
	}
	for key, want := range expectations {
		if got := headers[key]; got != want {
			t.Errorf("headers[%q] = %q, want %q", key, got, want)
		}
	}
	if headers["x-ots-signature"] == "" {
		t.Error("x-ots-signature missing")
	}
	if _, ok := headers["x-ots-ststoken"]; ok {
		t.Error("x-ots-ststoken should be absent without a token")
	}
}

func TestSortAndEncodeQuery(t *testing.T) {
	if got, want := sortAndEncodeQuery("b=2&a=1&a=0"), "a=0&a=1&b=2"; got != want {
		t.Errorf("sortAndEncodeQuery = %q, want %q", got, want)
	}
	if got := sortAndEncodeQuery(""); got != "" {
		t.Errorf("sortAndEncodeQuery(empty) = %q, want empty", got)
	}
}
