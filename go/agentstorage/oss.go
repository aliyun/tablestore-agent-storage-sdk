package agentstorage

import (
	"context"

	aliyunoss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSUploader uploads and deletes objects; implemented by the built-in
// aliyun-oss-go-sdk adapter and replaceable in tests.
type OSSUploader interface {
	PutObjectFromFile(ctx context.Context, objectKey, filePath string) error
	DeleteObjects(ctx context.Context, objectKeys []string) error
}

type ossConnection struct {
	endpoint        string
	bucket          string
	accessKeyID     string
	accessKeySecret string
	securityToken   string
}

func (c ossConnection) openBucket() (*aliyunoss.Bucket, error) {
	options := []aliyunoss.ClientOption{}
	if c.securityToken != "" {
		options = append(options, aliyunoss.SecurityToken(c.securityToken))
	}
	client, err := aliyunoss.New(c.endpoint, c.accessKeyID, c.accessKeySecret, options...)
	if err != nil {
		return nil, err
	}
	return client.Bucket(c.bucket)
}

// aliyunOSSUploader adapts aliyun-oss-go-sdk to OSSUploader.
type aliyunOSSUploader struct {
	ossConnection
}

func (u *aliyunOSSUploader) PutObjectFromFile(_ context.Context, objectKey, filePath string) error {
	bucket, err := u.openBucket()
	if err != nil {
		return err
	}
	return bucket.PutObjectFromFile(objectKey, filePath)
}

func (u *aliyunOSSUploader) DeleteObjects(_ context.Context, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}
	bucket, err := u.openBucket()
	if err != nil {
		return err
	}
	_, err = bucket.DeleteObjects(objectKeys, aliyunoss.DeleteObjectsQuiet(true))
	return err
}
