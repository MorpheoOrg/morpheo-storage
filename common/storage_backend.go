package dccommon

import (
	"bytes"
	"fmt"
	"io"

	"github.com/satori/go.uuid"
)

// StorageBackend describes the storage service API
type StorageBackend interface {
	GetData(id uuid.UUID) (dataReader io.Reader, err error)
	GetModel(id uuid.UUID) (modelReader io.Reader, err error)
}

// StorageAPIMock is a mock of the storage API (for tests & local dev. purposes)
type StorageAPIMock struct {
	StorageBackend

	evilDataUUID  string
	evilModelUUID string
}

// NewStorageAPIMock instantiates our mock of the storage API
func NewStorageAPIMock() (s *StorageAPIMock) {
	return &StorageAPIMock{
		evilDataUUID:  "58bc25d9-712d-4a53-8e73-2d6ca4d837c2",
		evilModelUUID: "610e134a-ff45-4416-aaac-1b3398e4bba6",
	}
}

// GetData returns fake data (the same, no matter the UUID)
func (s *StorageAPIMock) GetData(id uuid.UUID) (dataReader io.Reader, err error) {
	if id.String() == s.evilDataUUID {
		return nil, fmt.Errorf("Data %s not found on storage", id)
	}

	return bytes.NewBufferString("datamock"), nil
}

// GetModel returns a fake model, no matter the UUID
func (s *StorageAPIMock) GetModel(id uuid.UUID) (dataReader io.Reader, err error) {
	if id.String() == s.evilModelUUID {
		return nil, fmt.Errorf("Data %s not found on storage", id)
	}

	return bytes.NewBufferString("modelmock"), nil
}
