package agentstorage

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
)

// SignVersion selects the OTS signing protocol.
type SignVersion string

const (
	// SignV2 signs requests with HMAC-SHA1 (default, matches the Python/TS SDKs).
	SignV2 SignVersion = "v2"
	// SignV4 signs requests with HMAC-SHA256 using the OTS v4 derived key.
	// Requires Config.Region.
	SignV4 SignVersion = "v4"
)

const (
	v4SignatureProduct  = "ots"
	v4SignatureSeed     = "aliyun"
	v4SignaturePrefix   = v4SignatureSeed + "_v4"
	v4SignatureConstant = v4SignatureSeed + "_v4_request"
	v4SignatureSalt     = "ots"

	otsHeaderPrefix = "x-ots-"
)

func hmacSha1Base64(key []byte, content string) string {
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(content))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func hmacSha256Base64(key []byte, content string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func hmacSha256Bytes(key []byte, content string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

// sortAndEncodeQuery normalizes a raw query string: pairs sorted by key then
// value, re-encoded with URL encoding.
func sortAndEncodeQuery(queryString string) string {
	if queryString == "" {
		return ""
	}
	values, err := url.ParseQuery(queryString)
	if err != nil {
		return queryString
	}
	type pair struct{ key, value string }
	var pairs []pair
	for key, list := range values {
		for _, value := range list {
			pairs = append(pairs, pair{key, value})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].key != pairs[j].key {
			return pairs[i].key < pairs[j].key
		}
		return pairs[i].value < pairs[j].value
	})
	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(p.key))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(p.value))
	}
	return b.String()
}

func splitQuery(query string) (uri, queryString string) {
	if idx := strings.IndexByte(query, '#'); idx >= 0 {
		query = query[:idx]
	}
	if idx := strings.IndexByte(query, '?'); idx >= 0 {
		return query[:idx], query[idx+1:]
	}
	return query, ""
}

// makeHeadersString builds the canonical, sorted "key:value" lines of all
// x-ots-* headers that take part in the signature.
func makeHeadersString(headers map[string]string) string {
	var items []string
	for key, value := range headers {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, otsHeaderPrefix) {
			items = append(items, lower+":"+strings.TrimSpace(value))
		}
	}
	sort.Strings(items)
	return strings.Join(items, "\n")
}

func requestSignatureString(query string, headers map[string]string) string {
	uri, rawQuery := splitQuery(query)
	return uri + "\nPOST\n" + sortAndEncodeQuery(rawQuery) + "\n" + makeHeadersString(headers) + "\n"
}

// responseSignature reproduces the signature the server attaches to responses;
// both protocols verify it with HMAC-SHA1 over the derived signing key.
func responseSignature(query string, headers map[string]string, signingKey []byte) string {
	uri, _ := splitQuery(query)
	return hmacSha1Base64(signingKey, makeHeadersString(headers)+"\n"+uri)
}

// signer computes request signatures and exposes the signing key used to
// verify response signatures.
type signer interface {
	// signingKey returns the key used for the response signature check. The
	// signDate parameter is only meaningful for v4.
	signingKey(secret, signDate string) ([]byte, error)
	// addSignature computes the request signature over query+headers and adds
	// the protocol-specific x-ots-* signature headers. It returns the signDate
	// it used ("" for v2).
	addSignature(query string, headers map[string]string, secret string) (signDate string, err error)
}

type signerV2 struct{}

func (signerV2) signingKey(secret, _ string) ([]byte, error) {
	return []byte(secret), nil
}

func (s signerV2) addSignature(query string, headers map[string]string, secret string) (string, error) {
	headers["x-ots-signature"] = hmacSha1Base64([]byte(secret), requestSignatureString(query, headers))
	return "", nil
}

type signerV4 struct {
	region string
	// signDate overrides the auto-derived UTC date (yyyymmdd); used by tests.
	signDate string
	// now returns the current time formatted as yyyymmdd (UTC); nil means real time.
	now func() string
}

func (s *signerV4) currentSignDate() (string, error) {
	if s.signDate != "" {
		return s.signDate, nil
	}
	if s.now == nil {
		return "", newClientError("v4 sign_date is None.")
	}
	return s.now(), nil
}

func (s *signerV4) signingKey(secret, signDate string) ([]byte, error) {
	if signDate == "" {
		return nil, newClientError("v4 sign_date is None.")
	}
	first := hmacSha256Bytes([]byte(v4SignaturePrefix+secret), signDate)
	second := hmacSha256Bytes(first, s.region)
	third := hmacSha256Bytes(second, v4SignatureProduct)
	fourth := hmacSha256Bytes(third, v4SignatureConstant)
	// The OTS v4 protocol uses the base64 text of the derived key as the key bytes.
	return []byte(base64.StdEncoding.EncodeToString(fourth)), nil
}

func (s *signerV4) addSignature(query string, headers map[string]string, secret string) (string, error) {
	signDate, err := s.currentSignDate()
	if err != nil {
		return "", err
	}
	headers["x-ots-signdate"] = signDate
	headers["x-ots-signregion"] = s.region
	key, err := s.signingKey(secret, signDate)
	if err != nil {
		return "", err
	}
	headers["x-ots-signaturev4"] = hmacSha256Base64(key, requestSignatureString(query, headers)+v4SignatureSalt)
	return signDate, nil
}
