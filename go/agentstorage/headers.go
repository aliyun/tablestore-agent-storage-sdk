package agentstorage

import (
	"crypto/md5"
	"encoding/base64"
	"time"
)

const apiVersion = "2015-12-31"

func md5Base64(content []byte) string {
	sum := md5.Sum(content)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// buildDateHeader formats the x-ots-date header the way the Python SDK does:
// second precision with a fixed ".000Z" millisecond suffix.
func buildDateHeader(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05") + ".000Z"
}

type requestHeaderInput struct {
	body            []byte
	query           string
	instanceName    string
	accessKeyID     string
	accessKeySecret string
	securityToken   string
	signer          signer
	userAgent       string
	extraHeaders    map[string]string
	date            time.Time
}

// buildRequestHeaders assembles the signed x-ots-* headers plus User-Agent for
// one request. It returns the headers and the v4 signDate ("" for v2).
func buildRequestHeaders(in requestHeaderInput) (map[string]string, string, error) {
	date := in.date
	if date.IsZero() {
		date = time.Now()
	}
	headers := map[string]string{
		"x-ots-date":         buildDateHeader(date),
		"x-ots-apiversion":   apiVersion,
		"x-ots-accesskeyid":  in.accessKeyID,
		"x-ots-instancename": in.instanceName,
		"x-ots-contentmd5":   md5Base64(in.body),
	}
	if in.securityToken != "" {
		headers["x-ots-ststoken"] = in.securityToken
	}
	for key, value := range in.extraHeaders {
		headers[key] = value
	}
	signDate, err := in.signer.addSignature(in.query, headers, in.accessKeySecret)
	if err != nil {
		return nil, "", err
	}
	userAgent := in.userAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	headers["User-Agent"] = userAgent
	return headers, signDate, nil
}
