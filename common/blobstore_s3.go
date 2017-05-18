package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3BlobStore is a BlobStore implementations that stores data on AWS-S3
type S3BlobStore struct {
	session *s3Session
}

type s3Session struct {
	bucket StorageBucket
	s3     *s3.S3
	sess   *session.Session
}

// StorageBucket is the S3 bucket where data is stored
type StorageBucket struct {
	Name   string
	Region string
}

// Put streams a file to S3, given its size and uuid
func (s *S3BlobStore) Put(key string, r io.Reader, size int64) error {
	sess := s.session

	// Upload logic using a custom, presigned URL based, streaming uploader
	prereq, _ := sess.s3.PutObjectRequest(&s3.PutObjectInput{
		Bucket: &sess.bucket.Name,
		Key:    &key,
	})
	presignedURL, err := prereq.Presign(10 * time.Minute)
	if err != nil {
		return fmt.Errorf("[s3-storage] Error presigning request: %s", err)
	}

	req, err := http.NewRequest(http.MethodPut, presignedURL, r)
	req.ContentLength = size
	if err != nil {
		return fmt.Errorf("[s3-storage] Error constructing presigned request: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("[s3-storage] Error uploading file: %s", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("[s3-storage] Error reading S3 upload response body: %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("[s3-storage] Error uploading file (code: %d): %s", resp.StatusCode, buf.Bytes())
	}

	return nil
}

// Get retrieve a data with the specified uuid
func (s *S3BlobStore) Get(key string) (data io.ReadCloser, err error) {
	session := s.session
	file, err := session.s3.GetObject(&s3.GetObjectInput{
		Bucket: &session.bucket.Name,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return file.Body, err
}

func initWithBucket(bucket StorageBucket) (ret *s3Session) {
	ret = &s3Session{
		bucket: bucket,
		sess:   session.New(aws.NewConfig().WithRegion(bucket.Region)),
	}
	ret.s3 = s3.New(ret.sess)
	return ret
}

// NewStorageBucket creates a new StorageBucket
func NewStorageBucket(name, region string) StorageBucket {
	return StorageBucket{Name: name, Region: region}
}

// NewS3BlobStore creates a new S3Blobstore with default bucket
func NewS3BlobStore(awsBucket string, awsRegion string) (*S3BlobStore, error) {
	bucket := NewStorageBucket(awsBucket, awsRegion)
	s := new(S3BlobStore)
	s.session = initWithBucket(bucket)
	return s, nil
}
