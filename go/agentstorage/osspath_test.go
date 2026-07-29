package agentstorage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOssObjectKey(t *testing.T) {
	abs, err := filepath.Abs("/data/docs/file.pdf")
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.TrimLeft(filepath.ToSlash(abs), "/")

	got := buildOssObjectKey("inst", "kb", "", "/data/docs/file.pdf")
	want := "inst/kb/source_doc/_default/" + normalized
	if got != want {
		t.Errorf("key = %q, want %q", got, want)
	}

	got = buildOssObjectKey("inst", "kb", "sub", "/data/docs/file.pdf")
	if !strings.HasPrefix(got, "inst/kb/source_doc/sub/") {
		t.Errorf("key = %q, want subspace 'sub'", got)
	}
}

func TestBuildOssFullPath(t *testing.T) {
	if got, want := buildOssFullPath("bucket", "a/b.pdf"), "oss://bucket/a/b.pdf"; got != want {
		t.Errorf("full path = %q, want %q", got, want)
	}
}

func TestParseOssPath(t *testing.T) {
	bucket, key, ok := parseOssPath("oss://bucket/a/b.pdf", "default")
	if !ok || bucket != "bucket" || key != "a/b.pdf" {
		t.Errorf("parse = %q %q %v", bucket, key, ok)
	}

	bucket, key, ok = parseOssPath("bare/key.pdf", "default")
	if !ok || bucket != "default" || key != "bare/key.pdf" {
		t.Errorf("parse bare = %q %q %v", bucket, key, ok)
	}

	if _, _, ok := parseOssPath("oss://bucketonly", "default"); ok {
		t.Error("bucket without key should fail")
	}
	if _, _, ok := parseOssPath("bare/key.pdf", ""); ok {
		t.Error("bare key without default bucket should fail")
	}
}
