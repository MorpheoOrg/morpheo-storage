package common

import "io"

// BlobStore describes an form of storage targeted at storing files, regardless of the data they
// embed. A file is stored under a given key that can be used for further retrieval. It aims at
// abstracting disk storage as well as Amazon S3 (and alike) distributed storage platforms
type BlobStore interface {
	Put(key string, data io.Reader, size int64) error
	Get(key string) (data io.ReadCloser, err error)
}
