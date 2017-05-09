package common

import (
	"io"
	"os"
	"path/filepath"
)

// LocalBlobStore is a BlobStore implementations that stores data on the local hard drive
type LocalBlobStore struct {
	DataDir string
}

// Put writes a file in the data directory (and creates necessarry sub-directories if there are
// forward slashes in the key name)
func (s *LocalBlobStore) Put(key string, data io.Reader) error {
	datapath := filepath.Join(s.DataDir, key)

	parent := filepath.Dir(datapath)
	_, err := os.Stat(parent)
	if os.IsNotExist(err) {
		err := os.MkdirAll(parent, 0644)
		if err != nil {
			return err
		}
	}

	file, err := os.Create(datapath)
	defer file.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(file, data)
	return err
}

// Get returns an io.ReadCloser on the data living under the provided key. The retriever must
// explicitely call the Close() method on it when he's done reading.
func (s *LocalBlobStore) Get(key string) (data io.ReadCloser, err error) {
	datapath := filepath.Join(s.DataDir, key)
	return os.Open(datapath)
}
