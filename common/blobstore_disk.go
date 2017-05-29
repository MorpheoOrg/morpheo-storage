/*
 * Copyright Morpheo Org. 2017
 * 
 * contact@morpheo.co
 * 
 * This software is part of the Morpheo project, an open-source machine
 * learning platform.
 * 
 * This software is governed by the CeCILL license, compatible with the
 * GNU GPL, under French law and abiding by the rules of distribution of
 * free software. You can  use, modify and/ or redistribute the software
 * under the terms of the CeCILL license as circulated by CEA, CNRS and
 * INRIA at the following URL "http://www.cecill.info".
 * 
 * As a counterpart to the access to the source code and  rights to copy,
 * modify and redistribute granted by the license, users are provided only
 * with a limited warranty  and the software's author,  the holder of the
 * economic rights,  and the successive licensors  have only  limited
 * liability.
 * 
 * In this respect, the user's attention is drawn to the risks associated
 * with loading,  using,  modifying and/or developing or reproducing the
 * software by the user in light of its specific status of free software,
 * that may mean  that it is complicated to manipulate,  and  that  also
 * therefore means  that it is reserved for developers  and  experienced
 * professionals having in-depth computer knowledge. Users are therefore
 * encouraged to load and test the software's suitability as regards their
 * requirements in conditions enabling the security of their systems and/or
 * data to be ensured and,  more generally, to use and operate it in the
 * same conditions as regards security.
 * 
 * The fact that you are presently reading this means that you have had
 * knowledge of the CeCILL license and that you accept its terms.
 */

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

// NewLocalBlobStore creates a new local Blobstore given a data directory
func NewLocalBlobStore(dataDir string) (BlobStore, error) {
	return &LocalBlobStore{
		DataDir: dataDir,
	}, nil
}

// Put writes a file in the data directory (and creates necessarry sub-directories if there are
// forward slashes in the key name)
func (s *LocalBlobStore) Put(key string, data io.Reader, size int64) error {
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
