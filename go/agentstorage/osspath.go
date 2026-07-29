package agentstorage

import (
	"path/filepath"
	"strings"
)

const (
	defaultSubspace   = "_default"
	ossProtocolPrefix = "oss://"
	sourceDocFolder   = "source_doc"
)

func subspaceOrDefault(subspace string) string {
	if subspace == "" {
		return defaultSubspace
	}
	return subspace
}

// buildOssObjectKey derives the OSS object key an uploaded document lands at:
// {instance}/{kb}/source_doc/{subspace}/{absolute local path}.
func buildOssObjectKey(instanceName, knowledgeBaseName, subspace, filePath string) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	normalized := strings.TrimLeft(filepath.ToSlash(absPath), "/")
	return instanceName + "/" + knowledgeBaseName + "/" + sourceDocFolder + "/" +
		subspaceOrDefault(subspace) + "/" + normalized
}

// buildOssFullPath renders "oss://bucket/key".
func buildOssFullPath(bucketName, objectKey string) string {
	return ossProtocolPrefix + bucketName + "/" + objectKey
}

// parseOssPath splits "oss://bucket/key" into bucket and key. A path without
// the oss:// scheme is treated as a bare object key in defaultBucketName.
func parseOssPath(ossPath, defaultBucketName string) (bucketName, objectKey string, ok bool) {
	if !strings.HasPrefix(ossPath, ossProtocolPrefix) {
		if defaultBucketName == "" {
			return "", "", false
		}
		return defaultBucketName, ossPath, true
	}
	rest := ossPath[len(ossProtocolPrefix):]
	bucket, key, found := strings.Cut(rest, "/")
	if !found {
		return "", "", false
	}
	return bucket, key, true
}
